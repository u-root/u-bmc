// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type FanPlatform interface {
	PwmMap() map[int]string
	FanMap() map[int]string
	InitializeFans(*FanSystem) error
}

type FanSystem struct {
	fanMap map[int]string
	pwmMap map[int]string
}

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

func (f *FanSystem) ReadFanRpm(fan int) (int, error) {
	return readHwmon(f.fanMap, fan)
}

func (f *FanSystem) FanCount() int {
	return len(f.fanMap)
}

func (f *FanSystem) ReadFanPercentage(fan int) (int, error) {
	v, err := readHwmon(f.pwmMap, fan)
	if err != nil {
		return 0, err
	}
	return int(float32(v) * 100.0 / 255.0), nil
}

func (f *FanSystem) SetFanPercentage(fan int, prct int) error {
	v := int(float32(prct) * 255.0 / 100.0)
	return writeHwmon(f.pwmMap, fan, v)
}

func startFan(p FanPlatform) (*FanSystem, error) {
	f := FanSystem{fanMap: p.FanMap(), pwmMap: p.PwmMap()}
	err := p.InitializeFans(&f)
	if err != nil {
		return nil, fmt.Errorf("platform.InitializeFans: %v", err)
	}
	return &f, nil
}
