// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cert

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/u-root/u-bmc/config"
	"github.com/u-root/u-bmc/pkg/acme"
)

func Load(c *config.Acme, akey, fqdn, crt, key string) (*tls.Certificate, error) {
	return renewCert(c, akey, fqdn)
}

func loadOrGenerateKey(key string) (*ecdsa.PrivateKey, error) {
	b, err := ioutil.ReadFile(key)
	if os.IsNotExist(err) {
		return generateAndSaveKey(key)
	}
	return x509.ParseECPrivateKey(b)
}

func generateAndSaveKey(key string) (*ecdsa.PrivateKey, error) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	b, err := x509.MarshalECPrivateKey(k)
	if err != nil {
		return nil, err
	}
	return k, ioutil.WriteFile(key, b, 0400)
}

func renewCert(conf *config.Acme, akey, fqdn string) (*tls.Certificate, error) {
	key, err := loadOrGenerateKey(akey)

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
