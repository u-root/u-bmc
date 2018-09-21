// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/u-root/u-bmc/pkg/ast2400"
)

func configureSoc() {
	a := ast2400.Open()
	defer a.Close()

	// Configure UART routing:
	// - Route UART2 to UART3
	// - Route UART3 to UART2
	// TODO(bluecmd): Platform dependent
	a.Mem().MustWrite32(0x1E789000+0x9c, 0x6 << 22 | 0x4 << 19)

	// Re-enable the clock of UART2 to enable the internal routing
	// which will make u-bmc end of the pipe be /dev/ttyS2
	// This can be done by defining the uart2 as active in the dts, but
	// if we do that then /dev/ttyS1 might be confusing as it will not work
	// properly.
	a.Mem().MustWrite32(ast2400.SCU_BASE+0x0, ast2400.SCU_PASSWORD)
	csr := a.Mem().MustRead32(ast2400.SCU_BASE+0x0c)
	a.Mem().MustWrite32(ast2400.SCU_BASE+0x0c, csr & ^uint32(1 << 16))
	// Enable UART1 and UART2 pins
	mfr := a.Mem().MustRead32(ast2400.SCU_BASE+0x84)
	a.Mem().MustWrite32(ast2400.SCU_BASE+0x84, mfr | 0xffff0000)
	a.Mem().MustWrite32(ast2400.SCU_BASE+0x0, 0x0)
}
