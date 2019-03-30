// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"time"

	"github.com/u-root/u-bmc/pkg/aspeed"
	"github.com/u-root/u-bmc/pkg/bmc"
	"github.com/u-root/u-bmc/platform/quanta-f06-leopard-ddr3/pkg/platform"
)

func main() {
	p := platform.Platform()
	defer p.Close()

	a := aspeed.Open()
	defer a.Close()

	if err := bmc.Startup(p); err != nil {
		fmt.Printf("BOOT_TEST_FAILED: %v\n", err)
	}

	// Do nothing
	for {
		time.Sleep(10 * time.Second)
	}
}
