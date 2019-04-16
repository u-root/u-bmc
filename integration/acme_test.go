// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build amd64

package integration

import (
	"testing"
)

func TestACME(t *testing.T) {
	bmc, bmccleanup := BMCTest(t, &Options{
		Name: "TestACME-BMC",
		Cmds: []string{
			"github.com/u-root/u-root/cmds/init",
			"github.com/u-root/u-bmc/integration/testcmd/acme/uinit",
		},
	})
	defer bmccleanup()

	// If the system has booted that means the certificate was acquired
	if err := bmc.Expect("SYSTEM_BOOTED"); err != nil {
		t.Fatal(`expected "SYSTEM_BOOTED", got error: `, err)
	}
}
