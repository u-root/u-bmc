// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64

package integration

import (
	"os"
	"testing"
)

// TestBoot boots an image and then shuts down
func TestBoot(t *testing.T) {
	bmc, bmccleanup := BMCTest(t, 64, "../integration/testcmd/boot/*")
	defer bmccleanup()

	err := bmc.ConsoleExpect("TEST_OK")
	if err != nil {
		t.Fatalf("expected 'TEST_OK', got error: %v", err)
	}
}

// TestVerifyFail tries to boot an image with the wrong signature of /bbin/bb
// and expects the machines to stop and shut down
func TestVerifyFail(t *testing.T) {
	// Corrupt the signature by adding the contents of "/proc/uptime" at the end
	// when calculating the hash for /init (which is symlinked to /bin/bb).
	err := os.Setenv("__SIGN_EXTRA", "/proc/uptime")
	if err != nil {
		t.Fatalf("could not set ENV __SIGN_EXTRA: %v", err)
	}
	bmc, bmccleanup := BMCTest(t, 64, "../integration/testcmd/boot/*")
	defer bmccleanup()

	err = bmc.ConsoleExpect("invalid signature: hash tag doesn't match")
	if err != nil {
		t.Fatalf("expected 'invalid signature: hash tag doesnt match', got error: %v", err)
	}
	// Make sure the system rebooted in response to the error
	err = bmc.ConsoleExpect("Starting kernel ...")
	if err != nil {
		t.Fatalf("expected reboot signature 'Starting kernel ...', got error: %v", err)
	}
}
