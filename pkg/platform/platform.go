// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package platform

import (
	"github.com/u-root/u-bmc/pkg/ast2400"
)

var (
	linePortMap = map[string]uint32{
		// TODO(bluecmd): Extract from OpenBMC
		"DUMMY": ast2400.GpioPort("A0"),
	}

	// Reverse map of linePortMap
	portLineMap map[uint32]string
)

func init() {
	portLineMap = make(map[uint32]string)
	for k, v := range linePortMap {
		portLineMap[v] = k
	}
}

func LinePortMap() map[string]uint32 {
	// TODO(bluecmd): This will need to be abstracted away somehow if more
	// platforms are to be supported.
	return linePortMap
}

func GpioPortToName(p uint32) (string, bool) {
	s, ok := portLineMap[p]
	return s, ok
}
