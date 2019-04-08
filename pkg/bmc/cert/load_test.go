// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jmhodges/clock"
	"github.com/letsencrypt/pebble/ca"
	"github.com/letsencrypt/pebble/db"
	"github.com/letsencrypt/pebble/va"
	"github.com/letsencrypt/pebble/wfe"
	"github.com/spf13/afero"
	"github.com/u-root/u-bmc/config"
)

const (
	acmeKey  = "/config/acme.key"
	fqdn     = "ubmc.test"
	certFile = "/config/ubmc.test.crt"
	keyFile  = "/config/ubmc.test.key"
)

func TestAcme(t *testing.T) {
	cert := genCert()
	logger := Logger(t, "Pebble")
	clk := clock.New()
	db := db.NewMemoryStore(clk)
	ca := ca.New(logger, db)

	// Responding to challenges is tested in the integration test
	os.Setenv("PEBBLE_VA_ALWAYS_VALID", "1")
	os.Setenv("PEBBLE_VA_NOSLEEP", "1")

	// Enable strict mode to test upcoming API breaking changes
	strictMode := true
	va := va.New(logger, clk, 80, 443, strictMode)
	wfeImpl := wfe.New(logger, clk, db, va, ca, strictMode)
	muxHandler := wfeImpl.Handler()

	var tc tls.Config
	tc.Certificates = make([]tls.Certificate, 1)
	tc.Certificates[0] = cert

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatalf("net.Listen failed: %v", err)
	}

	go func() {
		tl := tls.NewListener(l, &tc)
		if err := http.Serve(tl, muxHandler); err != nil {
			log.Fatalf("http.Serve failed: %v", err)
		}
	}()

	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Certificate[0],
	}

	c := config.Acme{
		Directory:   fmt.Sprintf("https://%s/dir", l.Addr().String()),
		Contact:     "mailto:nobody@localhost",
		TermsAgreed: true,
		APICA:       string(pem.EncodeToMemory(block)),
	}

	fs := afero.NewMemMapFs()
	t.Run("TestInitialCert", func(t *testing.T) { testInitialCert(t, &c, fs) })
	t.Run("TestLoadCert", func(t *testing.T) { testLoadCert(t, &c, fs) })
}

func testInitialCert(t *testing.T, c *config.Acme, fs afero.Fs) {
	if _, err := load(fs, c, acmeKey, fqdn, certFile, keyFile); err != nil {
		t.Fatalf("Failed to load cert: %v", err)
	}
}

func testLoadCert(t *testing.T, c *config.Acme, fs afero.Fs) {
	if _, err := load(fs, c, "/acme-canary.key", fqdn, certFile, keyFile); err != nil {
		t.Fatalf("Failed to load cert: %v", err)
	}
	// If acme-canary.key was created the cert was not loaded from the file system
	ok, _ := afero.Exists(fs, "/acme-canary.key")
	if ok {
		t.Fatalf("ACME key was created which indicates that the certificate was not loaded from disk")
	}
}

func genCert() tls.Certificate {
	// From https://golang.org/src/crypto/tls/generate_cert.go
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("failed to generate private key: %s", err)
	}

	notBefore := time.Now().UTC()
	notAfter := notBefore.Add(time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"u-bmc Unit Test Company"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,

		IPAddresses: []net.IP{net.IPv6loopback, net.IPv4(127, 0, 0, 1)},
		DNSNames:    []string{"localhost"},
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatalf("x509.CreateCertificate: %v", err)
	}
	return tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  priv,
	}
}
