// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/u-root/u-bmc/config"
	"github.com/u-root/u-bmc/integration/util"
	"github.com/u-root/u-bmc/pkg/network/ttime"
	"github.com/u-root/u-bmc/pkg/system"
	"github.com/u-root/u-bmc/platform/qemu-virt-a72/pkg/platform"
)

func uinit() error {
	p := platform.Platform()
	defer p.Close()

	ca := util.NewTestCA()
	rt := util.NewTestRoughtimeServer()

	c := config.DefaultConfig
	c.RoughtimeServers = []ttime.RoughtimeServer{rt.Config}
	c.NtpServers = []ttime.NtpServer{}
	log.Printf("Roughtime server: %v", rt.Config)
	c.ACME.APICA = ca.APICA
	log.Printf("API CA: %v", ca.APICA)
	c.ACME.Directory = ca.Directory
	c.ACME.TermsAgreed = true

	err, sr := system.StartupWithConfig(p, c)
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
		util.FailTest(err)
	} else {
		util.PassTest()
	}
}
