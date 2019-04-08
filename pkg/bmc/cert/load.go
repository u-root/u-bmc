// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cert

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/spf13/afero"
	"github.com/u-root/u-bmc/config"
	"github.com/u-root/u-bmc/pkg/acme"
)

const (
	lifetimePadding = -4 * time.Hour
)

func Load(c *config.Acme, akey, fqdn, crt, key string) (*tls.Certificate, error) {
	return load(afero.NewOsFs(), time.Now(), c, akey, fqdn, crt, key)
}

func loadX509KeyPair(fs afero.Fs, certFile, keyFile string) (*tls.Certificate, error) {
	certPEMBlock, err := afero.ReadFile(fs, certFile)
	if err != nil {
		return nil, err
	}
	keyPEMBlock, err := afero.ReadFile(fs, keyFile)
	if err != nil {
		return nil, err
	}
	c, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
	return &c, err
}

func validateCert(ct *tls.Certificate, now time.Time) bool {
	c, err := x509.ParseCertificate(ct.Certificate[0])
	if err != nil {
		return false
	}
	// Check that the validity of the certificate is sane
	validFrom := c.NotBefore
	if validFrom.After(now) {
		return false
	}

	expires := c.NotAfter.Add(lifetimePadding)
	if expires.Before(now) {
		return false
	}
	return true
}

func load(fs afero.Fs, n time.Time, c *config.Acme, akey, fqdn, crt, key string) (*tls.Certificate, error) {
	kp, err := loadX509KeyPair(fs, crt, key)
	if err == nil && validateCert(kp, n) {
		return kp, nil
	}
	kp, err = renewCert(fs, c, akey, fqdn)
	if err != nil {
		return nil, err
	}

	// Fail soft on failing to write these files. We can still serve from memory
	// and next reboot we will have to renew instead.
	if err := saveKeyPair(fs, kp, crt, key); err != nil {
		log.Printf("WARNING: System will use certificate from memory, but will have to renew it on reboot. Error was: %v", err)
	}

	return kp, nil
}

func saveKeyPair(fs afero.Fs, kp *tls.Certificate, crt, key string) error {
	b, err := encodeCert(kp.Certificate)
	if err != nil {
		return fmt.Errorf("Failed to encode certificate, please file a bug about this: %v", err)
	} else {
		if err := afero.WriteFile(fs, crt, b, 0644); err != nil {
			return fmt.Errorf("Failed to save system certificate %s: %v", crt, err)
		}
	}

	b, err = encodeKey(kp.PrivateKey.(*ecdsa.PrivateKey))
	if err != nil {
		return fmt.Errorf("Failed to encode certificate, please file a bug about this")
	} else {
		if err := afero.WriteFile(fs, key, b, 0600); err != nil {
			return fmt.Errorf("Failed to save system certificate key %s: %v", key, err)
		}
	}
	return nil
}

func encodeCert(der [][]byte) ([]byte, error) {
	var res bytes.Buffer

	for _, c := range der {
		block := &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: c,
		}
		if err := pem.Encode(&res, block); err != nil {
			return []byte{}, err
		}
	}
	return res.Bytes(), nil
}

func encodeKey(pk *ecdsa.PrivateKey) ([]byte, error) {
	var res bytes.Buffer

	der, err := x509.MarshalECPrivateKey(pk)
	if err != nil {
		return []byte{}, err
	}

	block := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: der,
	}
	if err := pem.Encode(&res, block); err != nil {
		return []byte{}, err
	}
	return res.Bytes(), nil
}

func loadOrGenerateKey(fs afero.Fs, key string) (*ecdsa.PrivateKey, error) {
	b, err := afero.ReadFile(fs, key)
	if os.IsNotExist(err) {
		return generateAndSaveKey(fs, key)
	}
	return x509.ParseECPrivateKey(b)
}

func generateAndSaveKey(fs afero.Fs, key string) (*ecdsa.PrivateKey, error) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	b, err := x509.MarshalECPrivateKey(k)
	if err != nil {
		return nil, err
	}
	return k, afero.WriteFile(fs, key, b, 0400)
}

func renewCert(fs afero.Fs, conf *config.Acme, akey string, fqdn string) (*tls.Certificate, error) {
	key, err := loadOrGenerateKey(fs, akey)
	if err != nil {
		return nil, err
	}

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(conf.APICA))
	if !ok {
		return nil, fmt.Errorf("failed to parse Acme API CA certificate")
	}
	c := &acme.Client{
		Key:          key,
		DirectoryURL: conf.Directory,
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: roots,
				},
			},
		},
	}

	if err != nil {
		return nil, err
	}

	a := &acme.Account{
		Contact:     []string{conf.Contact},
		TermsAgreed: conf.TermsAgreed,
	}
	na, err := c.CreateAccount(context.Background(), a)
	if err != nil {
		return nil, err
	}

	if na.URL == "" {
		return nil, fmt.Errorf("empty na.URL")
	}

	order, err := c.CreateOrder(context.Background(), acme.NewOrder(fqdn))
	if err != nil {
		return nil, err
	}

	auth, err := c.GetAuthorization(context.Background(), order.Authorizations[0])
	if err != nil {
		return nil, err
	}

	var challenge *acme.Challenge
	for _, ch := range auth.Challenges {
		if ch.Type == "dns-01" {
			challenge = ch
			break
		}
	}
	if challenge == nil {
		return nil, fmt.Errorf("missing dns-01 challenge")
	}

	_, err = c.AcceptChallenge(context.Background(), challenge)
	if err != nil {
		return nil, err
	}

	_, err = c.WaitAuthorization(context.Background(), order.Authorizations[0])
	if err != nil {
		return nil, err
	}

	var cert tls.Certificate
	cert.PrivateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{DNSNames: []string{fqdn}}, cert.PrivateKey)
	if err != nil {
		return nil, err
	}

	der, err := c.FinalizeOrder(context.Background(), order.FinalizeURL, csr)
	if err != nil {
		return nil, err
	}

	cert.Certificate = der
	return &cert, nil
}
