// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64

package integration

import (
	"testing"
)

func TestACME(t *testing.T) {
	bmc, bmccleanup := BMCTest(t, 64, "../integration/testcmd/acme/*")
	defer bmccleanup()

	// If the system has booted that means the certificate was acquired
	err := bmc.ConsoleExpect("SYSTEM_BOOTED")
	if err != nil {
		t.Fatalf("expected 'SYSTEM_BOOTED', got error: %v", err)
	}
}
