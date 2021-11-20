// Copyright 2018-2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aspeed

import (
	"testing"
)

func TestDisableEnableWdt(t *testing.T) {
	fm := fakeMemory(t)
	a := OpenWithMemory(fm)
	fm.ExpectWrite32(0x1e78500c, 0)
	fm.ExpectWrite32(0x1e78502c, 0)
	a.DisableWdt()
	fm.ExpectWrite32(0x1e785034, 1)
	fm.ExpectWrite32(0x1e78502c, 0x82)
	a.EnableWdt()
}

func TestResetCpu(t *testing.T) {
	fm := fakeMemory(t)
	a := OpenWithMemory(fm)
	fm.ExpectWrite32(0x1e785004, 16)
	fm.ExpectWrite32(0x1e785008, 0x4755)
	fm.ExpectWrite32(0x1e78500c, 0x3)
	a.ResetCpu()
}
