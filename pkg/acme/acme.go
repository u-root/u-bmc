// Copyright 2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package acme

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"

	"github.com/caddyserver/certmagic"
	"github.com/u-root/u-bmc/config"
	"github.com/u-root/u-bmc/pkg/logger"
	"github.com/u-root/u-bmc/pkg/web"
)

var log = logger.LogContainer.GetLogger()

// ACMEConfig contains information about the ACME account
type ACMEConfig config.ACME

// GetManagedCert uses LetsEncrypt to obtain a valid TLS certificate and renews it automatically
func (c *ACMEConfig) GetManagedCert(fqdn []string, staging bool, serv *web.WebServer) (*tls.Config, error) {
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
	err := http.Serve(serv.Listener, acmeManager.HTTPChallengeHandler(serv.Mux))
	if err != nil {
		return nil, err
	}
	// Obtain and renew certificates
	err = acmeHandler.ManageSync(context.TODO(), fqdn)
	if err != nil {
		return nil, err
	}

	return acmeHandler.TLSConfig(), nil
}

func certPath() string {
	err := os.MkdirAll("/config/acme/", 0640)
	if err != nil {
		return "/tmp/"
	}
	return "/config/acme/"
}
