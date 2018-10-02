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
	tmpDir, q := testWithQEMU(t, "boot", "TestBoot", []string{})
	defer cleanup(t, tmpDir, q)

	if err := q.Expect("BOOT_TEST_OK"); err != nil {
		t.Fatal(`expected "BOOT_TEST_OK", got error: `, err)
	}
}

// TestVerifyFail tries to boot an image with the wrong signature of /bbin/bb
// and expects the machines to stop and shut down
func TestVerifyFail(t *testing.T) {
	// Corrupt the signature by adding the contents of "/proc/uptime" at the end
	// when calculating the hash for /init (which is symlinked to /bbin/bb).
	tmpDir, q := testWithQEMU(t, "boot", "TestVerifyFail", []string{"TEST_EXTRA_SIGN=/proc/uptime"})
	defer cleanup(t, tmpDir, q)

	if err := q.Expect("invalid signature: hash tag doesn't match"); err != nil {
		t.Fatal(`expected "invalid signature: hash tag doesn't match", got error: `, err)
	}
	// Make sure the system rebooted in response to the error
	if err := q.Expect("DRAM Init-DDR3"); err != nil {
		t.Fatal(`expected reboot signature "DRAM Init-DDR3", got error: `, err)
	}
}
