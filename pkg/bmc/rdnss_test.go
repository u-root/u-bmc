// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestRDNSSDecode(t *testing.T) {
	in := []byte{
		10, 64, 24, 0, 2, 0, 0, 0, 134, 0, 1, 0, 252, 0, 0, 0,
		25, 3, 0, 0, 0, 0, 1, 44, 252, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1,
	}
	opt, err := parseNDUserOpt(in)
	if err != nil {
		t.Fatalf("Error parsing neighbour discovery user options packet: %v", err)
	}

	if opt.Family != unix.AF_INET6 {
		t.Errorf("Expected ND user option to be for IPv6 (%d), was %d", unix.AF_INET6, opt.Family)
	}
	if opt.Ifindex != 2 {
		t.Errorf("Expected ND user option to be for interface 2, was %d", opt.Ifindex)
	}
	if opt.ICMPType != ND_ROUTER_ADVERT {
		t.Errorf("Expected ND user option to be ND_ROUTER_ADVERT (%d), was %d", ND_ROUTER_ADVERT, opt.ICMPType)
	}
	if opt.ICMPCode != 0 {
		t.Errorf("Expected ND user option to have ICMP code 0, had %d", opt.ICMPCode)
	}

	if len(opt.RDNSS) != 1 {
		t.Fatalf("Expected ND user option contain exactly one RDNSS entry, has %d", len(opt.RDNSS))
	}
	if len(opt.RDNSS[0].Server) != 1 {
		t.Fatalf("Expected RDNSS to have exactly one server, had %d", len(opt.RDNSS[0].Server))
	}
	if opt.RDNSS[0].Lifetime != time.Duration(300)*time.Second {
		t.Errorf("Expected RDNSS to have a 300 second lifetime, had %v", opt.RDNSS[0].Lifetime)
	}
	if opt.RDNSS[0].Server[0].String() != "fc00::1" {
		t.Errorf("Expected RDNSS to have fc00::1 as server, had %s", opt.RDNSS[0].Server[0])
	}
}

func TestRDNSSDecodeTooShort(t *testing.T) {
	in := []byte{
		10, 64, 24, 0, 2, 0, 0, 0, 134, 0, 1, 0, 252, 0, 0, 0,
		25, 3, 0, 0, 0, 0, 1, 44, 252, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	}
	_, err := parseNDUserOpt(in)
	if err == nil {
		t.Fatalf("Expected decode error not success")
	}
}

func TestRDNSSDecodeIgnore(t *testing.T) {
	in := []byte{
		10, 64, 24, 0, 2, 0, 0, 0, 134, 0, 0, 0, 8, 0, 15, 0,
		25, 0, 0, 0, 0, 0, 1, 44, 252, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1,
		20, 0, 1, 0, 254, 128, 0, 0, 0, 0, 0, 0, 2, 80, 86, 255, 254, 183, 140, 96,
	}
	opt, err := parseNDUserOpt(in)
	if err != nil {
		t.Fatalf("Error parsing neighbour discovery user options packet: %v", err)
	}
	if opt != nil {
		t.Fatalf("Expected RDNSS packet with 0 length to be ignored")
	}
}
