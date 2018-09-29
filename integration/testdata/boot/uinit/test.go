// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/u-root/u-bmc/pkg/ast2400"
	"github.com/u-root/u-bmc/pkg/bmc"
	"github.com/u-root/u-bmc/platform/quanta-f06-leopard-ddr3/pkg/platform"
	"golang.org/x/sys/unix"
)

func main() {
	p := platform.Platform()
	defer p.Close()

	a := ast2400.Open()
	defer a.Close()

	if err := bmc.Startup(p); err != nil {
		fmt.Printf("BOOT_TEST_FAILED: %v\n", err)
	} else {
		// Verify that the power button is set to an output for sanity
		s := a.SnapshotGpio()
		port, _ := p.GpioNameToPort("BMC_PWR_BTN_OUT_N")
		if !s.PortDirection(port) {
			fmt.Printf("BOOT_TEST_FAILED: BMC_PWR_BTN_OUT_N not output\n")
		} else {
			for {
				fmt.Printf("BOOT_TEST_OK\n")
			}
		}
	}
	unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF)
}
