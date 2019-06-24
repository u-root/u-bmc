// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build amd64

package integration

import (
	"testing"
)

// TestBoot boots an image and then shuts down
func TestBoot(t *testing.T) {
	bmc, bmccleanup := BMCTest(t, &Options{
		Name: "TestBoot-BMC",
		Cmds: []string{
			"github.com/u-root/u-root/cmds/core/init",
			"github.com/u-root/u-bmc/integration/testcmd/boot/uinit",
		},
	})
	defer bmccleanup()

	if err := bmc.Expect("TEST_OK"); err != nil {
		t.Fatal(`expected "TEST_OK", got error: `, err)
	}
}

// TestVerifyFail tries to boot an image with the wrong signature of /bbin/bb
// and expects the machines to stop and shut down
func TestVerifyFail(t *testing.T) {
	// Corrupt the signature by adding the contents of "/proc/uptime" at the end
	// when calculating the hash for /init (which is symlinked to /bbin/bb).
	bmc, bmccleanup := BMCTest(t, &Options{
		Name: "TestVerifyFail-BMC",
		Cmds: []string{
			"github.com/u-root/u-root/cmds/core/init",
			"github.com/u-root/u-bmc/integration/testcmd/boot/uinit",
		},
		ExtraBuildEnv: []string{"TEST_EXTRA_SIGN=/proc/uptime"},
	})
	defer bmccleanup()

	if err := bmc.Expect("invalid signature: hash tag doesn't match"); err != nil {
		t.Fatal(`expected "invalid signature: hash tag doesn't match", got error: `, err)
	}
	// Make sure the system rebooted in response to the error
	if err := bmc.Expect("Starting kernel ..."); err != nil {
		t.Fatal(`expected reboot signature "Starting kernel ...", got error: `, err)
	}
}
