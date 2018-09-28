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
	tmpDir, q := testWithQEMU(t, "boot", []string{})
	defer cleanup(t, tmpDir, q)

	if err := q.Expect("BOOT_TEST_OK"); err != nil {
		t.Fatal(`expected "BOOT_TEST_OK", got error: `, err)
	}
}
