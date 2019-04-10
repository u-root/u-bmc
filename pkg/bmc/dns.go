// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"log"
)

type dnsServer struct {
	fqdn string
}

func (s *dnsServer) HandleDNS01Challenge(fqdn string, record string) error {
	log.Printf("TODO: HandleDNS01Challenge %s TXT %s", fqdn, record)
	return nil
}

func startDNS(fqdn string) (*dnsServer, error) {
	return &dnsServer{fqdn}, nil
}
