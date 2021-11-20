// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/u-root/u-bmc/pkg/hardware/aspeed"
)

func main() {
	a := aspeed.Open()
	defer a.Close()

	a.SetResetControl(aspeed.SCU_DEFAULT_RESET)
	fmt.Printf("SCU04: %08x\n", a.Mem().MustRead32(aspeed.SCU_BASE+0x4))

	a.ResetCpu()
	a.UnfreezeCpu()
}
