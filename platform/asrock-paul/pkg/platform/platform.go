// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package platform

import (
	"github.com/u-root/u-bmc/pkg/hardware/aspeed"
	"github.com/u-root/u-bmc/pkg/hardware/gpio"
	pgpio "github.com/u-root/u-bmc/platform/aspeed-ast2500evb/pkg/gpio"
)

type platform struct {
	a *aspeed.Ast
	g *gpio.GpioSystem
	pgpio.Gpio
}

func (p *platform) InitializeGpio(g *gpio.GpioSystem) error {
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
	return "/dev/ttyS2", 115200
}

func (p *platform) Close() {
	p.a.Close()
}

func Platform() *platform {
	a := aspeed.Open()
	p := platform{a, nil, pgpio.Gpio{}}
	return &p
}
