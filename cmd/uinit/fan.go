// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

var (
	fanMap = map[int]string {
		0: "hwmon0/fan1_input",
		1: "hwmon0/fan2_input",
	}
	pwmMap = map[int]string {
		0: "hwmon0/pwm1",
		1: "hwmon0/pwm2",
	}
)

func readHwmon(m map[int]string, fan int) (int, error) {
	fname, ok := m[fan]
	if !ok {
		return 0, fmt.Errorf("No such fan %d", fan)
	}
	f, err := os.OpenFile("/sys/class/hwmon/" + fname, os.O_RDWR, 0600)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	b := make([]byte, 128)
	n, err := f.Read(b)
	if err != nil {
		return 0, err
	}
	v, err := strconv.Atoi(strings.Trim(string(b[:n]), "\n"))
	if err != nil {
		return 0, err
	}
	return v, nil
}

func writeHwmon(m map[int]string, fan int, v int) error {
	fname, ok := m[fan]
	if !ok {
		return fmt.Errorf("No such fan %d", fan)
	}
	f, err := os.OpenFile("/sys/class/hwmon/" + fname, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	f.Write([]byte(fmt.Sprintf("%d", v)))
	return nil
}

func readFanRpm(fan int) (int, error) {
	return readHwmon(fanMap, fan)
}

func fanCount() int {
	return len(fanMap)
}

func readFanPercentage(fan int) (int, error) {
	v, err := readHwmon(pwmMap, fan)
	if err != nil {
		return 0, err
	}
	return int(float32(v) * 100.0 / 255.0), nil
}

func setFanPercentage(fan int, prct int) error {
	v := int(float32(prct) * 255.0 / 100.0)
	return writeHwmon(pwmMap, fan, v)
}
