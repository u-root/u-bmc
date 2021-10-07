// Copyright 2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package platform

import (
	"github.com/u-root/u-bmc/pkg/bmc"
	"github.com/u-root/u-bmc/platform/qemu-virt-a15/pkg/gpio"
)

type platform struct {
	g *bmc.GpioSystem
	gpio.Gpio
}

func (p *platform) InitializeGpio(g *bmc.GpioSystem) error {
	return nil
}

func (p *platform) InitializeSystem() error {
	return nil
}

func (p *platform) PwmMap() map[int]string {
	return nil
}

func (p *platform) FanMap() map[int]string {
	return nil
}

func (p *platform) ThermometerMap() map[int]string {
	return nil
}

func (p *platform) HostUart() (string, int) {
	return "/dev/ttyAMA0", 115200
}

func (p *platform) Close() {
	return
}

func Platform() *platform {
	p := platform{nil, gpio.Gpio{}}
	return &p
}
