// Copyright 2019 the u-root Authors. All rights reserved
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
	return map[int]string{
		0: "/sys/class/hwmon/hwmon0/pwm1",
	}
}

func (p *platform) FanMap() map[int]string {
	return map[int]string{
		0: "/sys/class/hwmon/hwmon0/fan1_input",
	}
}

func (p *platform) ThermometerMap() map[int]string {
	return map[int]string{
		0: "/sys/class/hwmon/hwmon1/temp1_input",
	}
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
