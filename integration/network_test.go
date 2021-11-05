// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64

package integration

import (
	"testing"
)

// Tries to access /metrics from a remote host
func TestMetrics(t *testing.T) {
	bmc, bmccleanup := BMCTest(t, 64, "../integration/testcmd/metrics/*")
	defer bmccleanup()

	err := bmc.ConsoleExpect("TEST_OK")
	if err != nil {
		t.Fatalf("expected 'TEST_OK' on host, got error: %v", err)
	}
}
