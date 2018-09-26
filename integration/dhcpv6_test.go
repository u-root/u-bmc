// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build amd64

package integration

import (
	"testing"
)

// TestDhcpv6 boots an image, sets up a DHCPv6 server, gets a lease
// and tries to ping the address. If that works, it's all OK.
func TestDhcpv6(t *testing.T) {
	tmpDir, q := testWithQEMU(t, "dhcpv6", []string{})
	defer cleanup(t, tmpDir, q)

	if err := q.Expect("DHCPV6_TEST_OK"); err != nil {
		t.Fatal(`expected "DHCPV6_TEST_OK", got error: `, err)
	}
}
