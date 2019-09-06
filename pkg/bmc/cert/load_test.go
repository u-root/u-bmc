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

	"github.com/letsencrypt/pebble/ca"
	"github.com/letsencrypt/pebble/db"
	"github.com/letsencrypt/pebble/va"
	"github.com/letsencrypt/pebble/wfe"
	"github.com/spf13/afero"
	"github.com/u-root/u-bmc/config"
)

type fakeACMEHandler struct {
}

func (h *fakeACMEHandler) HandleDNS01Challenge(string, string) error {
	return nil
}

// TODO(bluecmd): Disabled because it's flaky on CircleCI
func TestACME(t *testing.T) {
	t.Skip("TestACME is disabled because flaky CircleCI")
	cert := genCert()
	logger := Logger(t, "Pebble")
	db := db.NewMemoryStore()
	ca := ca.New(logger, db, "", 0)

	// Responding to challenges is tested in the integration test
	os.Setenv("PEBBLE_VA_ALWAYS_VALID", "1")
	os.Setenv("PEBBLE_VA_NOSLEEP", "1")

	// Enable strict mode to test upcoming API breaking changes
	strictMode := true
	va := va.New(logger, 80, 443, strictMode, "")
	wfeImpl := wfe.New(logger, db, va, ca, strictMode)
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

	fs := afero.NewMemMapFs()
	pk, _ := loadOrGenerateKey(fs, "account.key")

	m := &Manager{
		FQDN:         "ubmc.test",
		AccountKey:   pk,
		ACMEHandlers: []ACMEHandler{&fakeACMEHandler{}},
		ACMEConfig: &config.ACME{
			Directory:   fmt.Sprintf("https://%s/dir", l.Addr().String()),
			Contact:     "mailto:nobody@localhost",
			TermsAgreed: true,
			APICA:       string(pem.EncodeToMemory(block)),
		},
	}

	now := time.Now()
	kp, err := m.maybeRenew(now, nil)
	if err != nil {
		t.Fatalf("Failed to load cert: %v", err)
	}

	// Try to renew just a bit after
	now = now.AddDate(0, 0, 1)
	kp2, err := m.maybeRenew(now, kp)
	if err != nil {
		t.Fatalf("Failed to load cert: %v", err)
	}

	if kp != kp2 {
		t.Fatalf("Certificate changed when it should not have, %v != %v", kp, kp2)
	}

	// Pebble mints 5 year certificates by default
	// Pebble sometimes re-use the ID and fails, so let's retry
	for i := 0; i < 5; i++ {
		now = now.AddDate(5, 0, 0)
		kp2, err = m.maybeRenew(now, kp)
		if err == nil {
			break
		}
		t.Logf("Failed to load cert: %v, retrying", err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		t.Fatalf("Failed to load cert: %v", err)
	}
	if kp == kp2 {
		t.Fatalf("Certificate remained the same when it should have been renewed")
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
