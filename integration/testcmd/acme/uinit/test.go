// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/u-root/u-bmc/config"
	"github.com/u-root/u-bmc/integration/utils"
	"github.com/u-root/u-bmc/pkg/bmc"
	"github.com/u-root/u-bmc/pkg/bmc/ttime"
	"github.com/u-root/u-bmc/platform/quanta-f06-leopard-ddr3/pkg/platform"
)

func uinit() error {
	p := platform.Platform()
	defer p.Close()

	// TODO(bluecmd): Test with validation
	os.Setenv("PEBBLE_VA_ALWAYS_VALID", "1")

	ca := utils.NewTestCA()
	rt := utils.NewTestRoughtimeServer()

	c := config.DefaultConfig
	c.RoughtimeServers = []ttime.RoughtimeServer{rt.Config}
	c.NtpServers = []ttime.NtpServer{}
	log.Printf("Roughtime server: %v", rt.Config)
	c.ACME.APICA = ca.APICA
	log.Printf("API CA: %v", ca.APICA)
	c.ACME.Directory = ca.Directory
	c.ACME.TermsAgreed = true

	err, sr := bmc.StartupWithConfig(p, c)
	if err != nil {
		return err
	}

	// Network has been configured, start helper servers
	go ca.Run()
	go rt.Run()

	if err := <-sr; err != nil {
		return err
	}

	fmt.Println("SYSTEM_BOOTED")

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
