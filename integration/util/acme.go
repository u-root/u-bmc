// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/jmhodges/clock"
	"github.com/letsencrypt/pebble/ca"
	"github.com/letsencrypt/pebble/db"
	"github.com/letsencrypt/pebble/va"
	"github.com/letsencrypt/pebble/wfe"
)

type CAServer struct {
	APICA     string
	Directory string

	cert    *tls.Certificate
	handler http.Handler
}

func NewTestCA() *CAServer {
	cert := genCert()
	clk := clock.New()

	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			d := net.Dialer{}
			return d.DialContext(ctx, "udp", "[::1]:53")
		},
	}

	logger := log.New(os.Stdout, "Pebble ", log.LstdFlags)
	db := db.NewMemoryStore(clk)
	ca := ca.New(logger, db)

	// Enable strict mode to test upcoming API breaking changes
	strictMode := true
	va := va.New(logger, clk, 80, 443, strictMode)
	wfeImpl := wfe.New(logger, clk, db, va, ca, strictMode)
	muxHandler := wfeImpl.Handler()

	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: []byte(cert.Certificate[0]),
	}

	return &CAServer{
		APICA:     string(pem.EncodeToMemory(block)),
		Directory: "https://[::1]:14000/dir",
		cert:      &cert,
		handler:   muxHandler,
	}
}

func (c *CAServer) Run() {
	var config tls.Config
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0] = *c.cert

	l, err := net.Listen("tcp", ":14000")
	if err != nil {
		log.Fatalf("net.Listen failed: %v", err)
	}

	tl := tls.NewListener(l, &config)
	if err := http.Serve(tl, c.handler); err != nil {
		log.Fatalf("http.Serve failed: %v", err)
	}
}

func genCert() tls.Certificate {
	// From https://golang.org/src/crypto/tls/generate_cert.go
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("failed to generate private key: %s", err)
	}

	// The current time when the cert is generated is 1970 since it's before
	// the time has been acquired. Make sure that the certificate is valid
	// for a loooong time.
	notBefore := time.Now().UTC()
	// TODO(bluecmd): Golang's ASN.1 marshaller seems to be having issues with
	// dates past 2038.
	notAfter := time.Unix(1<<31-1, 0).UTC()

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		log.Fatalf("failed to generate serial number: %s", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"u-bmc Integration Test Company"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,

		IPAddresses: []net.IP{net.IPv6loopback},
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
