// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"time"

	"github.com/u-root/u-bmc/pkg/hardware/aspeed"
	"github.com/u-root/u-bmc/pkg/system"
	"github.com/u-root/u-bmc/platform/qemu-virt-a72/pkg/platform"
	"golang.org/x/sys/unix"
)

func main() {
	p := platform.Platform()
	defer p.Close()

	a := aspeed.Open()
	defer a.Close()

	if err, _ := system.Startup(p); err != nil {
		fmt.Printf("TEST_FAILED: %v\n", err)
	} else {
		// Verify that the power button is set to an output for sanity
		time.Sleep(3 * time.Second)
		s := a.SnapshotGpio()
		port, _ := p.GpioNameToPort("BMC_PWR_BTN_OUT_N")
		if !s.PortDirection(port) {
			fmt.Printf("TEST_FAILED: BMC_PWR_BTN_OUT_N not output\n")
		} else {
			for {
				fmt.Printf("TEST_OK\n")
			}
		}
	}
	unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF)
}
