// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io/ioutil"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/u-root/u-bmc/integration/utils"
	"github.com/u-root/u-bmc/pkg/bmc"
	"github.com/u-root/u-bmc/platform/quanta-f06-leopard-ddr3/pkg/platform"
)

type TempPlatform interface {
	ThermometerMap() map[int]string
}

func verifyTemperature(p TempPlatform, i int, celcius int) error {
	b, err := ioutil.ReadFile(p.ThermometerMap()[i])
	if err != nil {
		return err
	}

	millitemp, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return err
	}
	if math.Abs(float64(millitemp-celcius*1000)) > 3000 {
		return fmt.Errorf("Expected temperature to be %v +/- 3 C, was %v", celcius, millitemp)
	}
	return nil
}

func uinit() error {
	p := platform.Platform()
	defer p.Close()

	if err := bmc.Startup(p); err != nil {
		return err
	}

	fmt.Println("TEST_SET_NORMAL_TEMP")
	time.Sleep(3 * time.Second)
	if err := verifyTemperature(p, 0, 23); err != nil {
		return err
	}
	fmt.Println("TEST_SET_HIGH_TEMP")
	time.Sleep(3 * time.Second)
	if err := verifyTemperature(p, 0, 100); err != nil {
		return err
	}
	fmt.Println("TEST_SET_LOW_TEMP")
	time.Sleep(3 * time.Second)
	if err := verifyTemperature(p, 0, 0); err != nil {
		return err
	}
	return nil
}

func main() {
	if err := uinit(); err != nil {
		utils.FailTest(err)
	} else {
		utils.PassTest()
	}
}
