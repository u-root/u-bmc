// Copyright 2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package acme

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/u-root/u-bmc/config"
	"github.com/u-root/u-bmc/pkg/logger"
)

var log = logger.LogContainer.GetLogger()

// ACMEConfig contains information about the ACME account
type ACMEConfig struct {
	config.ACME
}

// GetManagedCert uses LetsEncrypt to obtain a valid TLS certificate and renews it automatically
func (c *ACMEConfig) GetManagedCert(fqdn []string, staging bool, mux *http.ServeMux) (*tls.Config, error) {
	// Create a certmagic config
	conf := &certmagic.Config{
		Logger:   log,
		Storage:  &certmagic.FileStorage{Path: certPath()},
		OnDemand: &certmagic.OnDemandConfig{},
	}
	// Create a cache to serve certificates from memory
	cache := certmagic.NewCache(certmagic.CacheOptions{
		GetConfigForCert: func(c certmagic.Certificate) (*certmagic.Config, error) {
			return conf, nil
		},
		Logger: log,
	})
	// Create a handler to later manage the certificates
	acmeHandler := certmagic.New(cache, *conf)
	// Create a pool of known root CA certificates
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM([]byte(c.APICA))
	// Create ACME manager that holds account and solver information
	var caDirectory string
	if staging {
		caDirectory = certmagic.LetsEncryptStagingCA
	} else {
		caDirectory = certmagic.LetsEncryptProductionCA
	}
	acmeManager := certmagic.NewACMEManager(acmeHandler, certmagic.ACMEManager{
		CA:           caDirectory,
		Email:        c.Contact,
		Agreed:       c.TermsAgreed,
		DNS01Solver:  StubProvider(),
		TrustedRoots: pool,
		Logger:       log,
	})
	// Add manager to handler and start HTTP01 solver
	acmeHandler.Issuers = []certmagic.Issuer{acmeManager}
	http.ListenAndServe(":80", acmeManager.HTTPChallengeHandler(mux))
	// Obtain and renew certificates
	err := acmeHandler.ManageSync(context.TODO(), fqdn)
	if err != nil {
		return nil, err
	}

	return acmeHandler.TLSConfig(), nil
}

// GetSelfSignedCert generates a self signed TLS certificate
func (c *ACMEConfig) GetSelfSignedCert(fqdn []string) (*tls.Config, error) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1337),
		Subject: pkix.Name{
			Organization: []string{"u-bmc local CA"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(5, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caPrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}
	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	caPrivkeyBytes, err := x509.MarshalECPrivateKey(caPrivKey)
	if err != nil {
		return nil, err
	}
	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: caPrivkeyBytes,
	})

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(4200),
		Subject: pkix.Name{
			Organization: []string{"u-bmc local CA"},
		},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:     fqdn,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(5, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 5},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	certPrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}
	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})
	certPrivKeyBytes, err := x509.MarshalECPrivateKey(certPrivKey)
	if err != nil {
		return nil, err
	}
	certPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: certPrivKeyBytes,
	})

	serverCert, err := tls.X509KeyPair(certPEM.Bytes(), certPrivKeyPEM.Bytes())
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{serverCert},
	}, nil
}

func certPath() string {
	err := os.MkdirAll("/config/acme", 0640)
	if err != nil {
		return "/tmp"
	}
	return "/config/acme"
}
