// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"fmt"
	"log"

	"golang.org/x/sys/unix"
)

func FailTest(err error) {
	log.Printf("Test failed with error: %v", err)
	fmt.Printf("TEST_FAIL")
	unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF)
	log.Fatalf("Test failed")
}

func PassTest() {
	fmt.Printf("TEST_OK")
	unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF)
	log.Fatalf("Test passed")
}
