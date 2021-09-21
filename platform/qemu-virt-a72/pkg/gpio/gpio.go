// Copyright 2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpio

var (
	linePortMap = map[string]uint32{}

	// Reverse map of linePortMap
	portLineMap map[uint32]string
)

type Gpio struct {
}

func init() {
	portLineMap = make(map[uint32]string)
	for k, v := range linePortMap {
		portLineMap[v] = k
	}
}

func (_ *Gpio) GpioNameToPort(l string) (uint32, bool) {
	s, ok := linePortMap[l]
	return s, ok
}

func (_ *Gpio) GpioPortToName(i uint32) (string, bool) {
	s, ok := portLineMap[i]
	return s, ok
}
