// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"time"

	"github.com/u-root/u-bmc/integration/util"
	"github.com/u-root/u-bmc/pkg/bmc"
	"github.com/u-root/u-bmc/platform/qemu-virt-a72/pkg/platform"
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
		util.FailTest(err)
	} else {
		util.PassTest()
	}
}
