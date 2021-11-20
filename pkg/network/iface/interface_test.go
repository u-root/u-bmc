// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package iface

import (
	"net"
	"testing"
)

func TestIPv6MACCheck(t *testing.T) {
	ip_ok := net.ParseIP("fe80::5054:ff:fe12:3456")
	ip_wrong := net.ParseIP("fec0::5054:ff:fe12:3456")
	mac_ok, _ := net.ParseMAC("52:54:00:12:34:56")
	mac_wrong, _ := net.ParseMAC("52:54:00:12:34:57")

	if !isLinkLocalForMAC(ip_ok.To16(), mac_ok) {
		t.Errorf("Expected %v to be the MAC for link-local IP %v", mac_ok, ip_ok)
	}
	if isLinkLocalForMAC(ip_wrong.To16(), mac_ok) {
		t.Errorf("Expected IP %v to not be link-local", ip_wrong)
	}
	if isLinkLocalForMAC(ip_ok.To16(), mac_wrong) {
		t.Errorf("Expected %v to not be the MAC for link-local IP %v", mac_wrong, ip_ok)
	}
}
