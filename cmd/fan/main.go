// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"

	"github.com/u-root/u-bmc/pkg/ast2400"
)

var (
	fan   = flag.Int("fan", -1, "Which fan to set, -1 to not set fan speed")
	speed = flag.Int("speed", 128, "Value between 0 < x < 255 where 255 is max speed.")
)

func main() {
	flag.Parse()

	a := ast2400.Open()
	defer a.Close()
	a.DumpPwm()
	fmt.Printf("Fan 0: %v RPM\n", a.MeasureFanRpm(0))
	fmt.Printf("Fan 1: %v RPM\n", a.MeasureFanRpm(2))

	if *fan > -1 {
		fmt.Printf("Setting fan %v to %v\n", *fan, *speed)
		a.SetFanDutyCycle(uint(*fan), uint8(*speed))
	}
}
