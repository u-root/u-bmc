// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/u-root/u-bmc/pkg/ast2400"
)

func main() {
	a := ast2400.Open()
	defer a.Close()

	a.SetResetControl(ast2400.SCU_DEFAULT_RESET)
	fmt.Printf("SCU04: %08x\n", a.Mem().MustRead32(ast2400.SCU_BASE+0x4))

	a.ResetCpu()
	a.UnfreezeCpu()
}
