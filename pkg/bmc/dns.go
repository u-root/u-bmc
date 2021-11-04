// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"net"
	"time"

	"github.com/miekg/dns"
)

type Addresser interface {
	IPv4() net.IP
	IPv6() net.IP
	AddressLifetime() time.Duration
}

type dnsServer struct {
	zone string
	addr Addresser
	chal *dns.TXT
}

func (s *dnsServer) HandleDNS01Challenge(fqdn string, record string) error {
	rrx := new(dns.TXT)
	rrx.Hdr = dns.RR_Header{
		Name:   fqdn + ".",
		Rrtype: dns.TypeTXT,
		Class:  dns.ClassINET,
		Ttl:    60,
	}
	rrx.Txt = []string{record}
	s.chal = rrx
	return nil
}

func (s *dnsServer) Reply(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	if len(r.Question) == 1 && r.Question[0].Name == s.zone {
		// TODO(bluecmd): Respond with A/AAAA for configured addresses
		m.SetRcode(r, dns.RcodeNameError)
	} else if len(r.Question) == 1 && s.chal != nil && r.Question[0].Name == s.chal.Hdr.Name {
		m.Answer = []dns.RR{s.chal}
	} else {
		m.SetRcode(r, dns.RcodeNameError)
	}
	if err := w.WriteMsg(m); err != nil {
		log.Errorf("DNS WriteMsg failed: %v", err)
	}
}

func startDNS(fqdn string, a Addresser) (*dnsServer, error) {
	s := &dnsServer{
		zone: fqdn + ".",
		addr: a,
	}
	dns.HandleFunc(fqdn+".", s.Reply)

	go func() {
		s := &dns.Server{Addr: ":53", Net: "udp"}
		if err := s.ListenAndServe(); err != nil {
			log.Errorf("DNS server failed (udp): %v", err)
		}
	}()
	go func() {
		s := &dns.Server{Addr: ":53", Net: "tcp"}
		if err := s.ListenAndServe(); err != nil {
			log.Errorf("DNS server failed (tcp): %v", err)
		}
	}()
	return s, nil
}
