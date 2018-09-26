// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/u-root/u-bmc/pkg/bmc"
	"github.com/u-root/u-bmc/platform/quanta-f06-leopard-ddr3/pkg/platform"
	"golang.org/x/sys/unix"
)

func test() error {
	// Enable ICMP Echo sockets
	f, err := os.OpenFile("/proc/sys/net/ipv4/ping_group_range", os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("Could not open ICMP Echo sockets proc file: %v", err)
	}
	defer f.Close()
	if _, err := f.Write([]byte("0\t1\n")); err != nil {
		return fmt.Errorf("Could not enable ICMP Echo sockets: %v", err)
	}
	p := platform.Platform()
	defer p.Close()
	if err := bmc.Startup(p); err != nil {
		return fmt.Errorf("bmc.Startup: %v", err)
	}
	if err := ping("ff02::1", "eth0"); err != nil {
		return fmt.Errorf("ping: %v", err)
	}
	return nil
}

func main() {
	if err := test(); err != nil {
		log.Printf("DHCPV6_TEST_FAILED: %v", err)
	} else {
		log.Printf("BOOT_TEST_OK")
	}
	unix.Reboot(unix.LINUX_REBOOT_CMD_POWER_OFF)
}
