// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"time"

	"github.com/u-root/u-bmc/integration/utils"
	"github.com/u-root/u-bmc/pkg/bmc"
	"github.com/u-root/u-bmc/platform/quanta-f06-leopard-ddr3/pkg/platform"
)

func uinit() error {
	p := platform.Platform()
	defer p.Close()

	err, sr := bmc.Startup(p)

	if err != nil {
		return err
	}

	if err := <-sr; err != nil {
		return err
	}

	// Hang around forever
	for {
		time.Sleep(10 * time.Second)
	}
}

func main() {
	if err := uinit(); err != nil {
		utils.FailTest(err)
	} else {
		utils.PassTest()
	}
}
