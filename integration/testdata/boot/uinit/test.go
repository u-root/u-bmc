// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/u-root/u-bmc/pkg/bmc"
	"github.com/u-root/u-bmc/platform/quanta-f06-leopard-ddr3/pkg/platform"
	"golang.org/x/sys/unix"
)

func main() {
	p := platform.Platform()
	defer p.Close()
	if err := bmc.Startup(p); err != nil {
		fmt.Printf("BOOT_TEST_FAILED: %v\n", err)
	} else {
		fmt.Printf("BOOT_TEST_OK\n")
	}
	unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF)
}
