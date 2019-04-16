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

type ACMEHandler interface {
	HandleDNS01Challenge(fqdn string, record string) error
}

type Manager struct {
	FQDN         string
	AccountKey   *ecdsa.PrivateKey
	ACMEConfig   *config.ACME
	ACMEHandlers []ACMEHandler
}

func (m *Manager) MaybeRenew(kp *tls.Certificate) (*tls.Certificate, error) {
	return m.maybeRenew(time.Now(), kp)
}

func validateCert(ct *tls.Certificate, now time.Time) bool {
	if ct == nil {
		return false
	}
	if len(ct.Certificate) == 0 {
		return false
	}

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

func (m *Manager) maybeRenew(n time.Time, kp *tls.Certificate) (*tls.Certificate, error) {
	// Check if there is a need to renew the cert, skip otherwise
	if validateCert(kp, n) {
		return kp, nil
	}
	kp, err := m.renew()
	if err != nil {
		return nil, err
	}

	return kp, nil
}

func SaveKeyPair(kp *tls.Certificate, crt, key string) error {
	return saveKeyPair(afero.NewOsFs(), kp, crt, key)
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

func LoadOrGenerateKey(file string) (*ecdsa.PrivateKey, error) {
	return loadOrGenerateKey(afero.NewOsFs(), file)
}

func loadOrGenerateKey(fs afero.Fs, file string) (*ecdsa.PrivateKey, error) {
	b, err := afero.ReadFile(fs, file)
	if os.IsNotExist(err) {
		return generateAndSaveKey(fs, file)
	}
	return x509.ParseECPrivateKey(b)
}

func generateAndSaveKey(fs afero.Fs, file string) (*ecdsa.PrivateKey, error) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	b, err := x509.MarshalECPrivateKey(k)
	if err != nil {
		return nil, err
	}
	return k, afero.WriteFile(fs, file, b, 0400)
}

func (m *Manager) renew() (*tls.Certificate, error) {
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(m.ACMEConfig.APICA))
	if !ok {
		return nil, fmt.Errorf("failed to parse Acme API CA certificate")
	}
	c := &acme.Client{
		Key:          m.AccountKey,
		DirectoryURL: m.ACMEConfig.Directory,
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs: roots,
				},
			},
		},
	}

	a := &acme.Account{
		Contact:     []string{m.ACMEConfig.Contact},
		TermsAgreed: m.ACMEConfig.TermsAgreed,
	}
	na, err := c.CreateAccount(context.Background(), a)
	if err != nil {
		return nil, err
	}

	if na.URL == "" {
		return nil, fmt.Errorf("empty na.URL")
	}

	order, err := c.CreateOrder(context.Background(), acme.NewOrder(m.FQDN))
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

	t, err := c.DNS01ChallengeRecord(challenge.Token)
	if err != nil {
		return nil, err
	}

	handled := false
	for _, h := range m.ACMEHandlers {
		if err := h.HandleDNS01Challenge("_acme-challenge."+m.FQDN, t); err == nil {
			handled = true
			break
		}
	}
	if !handled {
		return nil, fmt.Errorf("No DNS01 ACME handler could handle challenge")
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

	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{DNSNames: []string{m.FQDN}}, cert.PrivateKey)
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
