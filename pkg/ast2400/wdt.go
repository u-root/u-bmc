// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ast2400

const (
	WDT1_RLD_CTR   uintptr = 0x1e785004
	WDT1_RESTART   uintptr = 0x1e785008
	WDT1_CTRL      uintptr = 0x1e78500c
	WDT2_CTRL      uintptr = 0x1e78502c
	WDT2_TMOUT_CLR uintptr = 0x1e785034

	WDT_RESTART_PASSWORD uint32 = 0x4755
)

func (a *Ast) DisableWdt() {
	a.Mem().MustWrite32(WDT1_CTRL, 0)
	a.Mem().MustWrite32(WDT2_CTRL, 0)
}

func (a *Ast) EnableWdt() {
	// 0x1 - Reset boot code source select
	a.Mem().MustWrite32(WDT2_TMOUT_CLR, 0x1)
	// 0x80 - Use second boot code whenever WDT reset
	// 0x2  - Reset system after timeout
	a.Mem().MustWrite32(WDT2_CTRL, 0x82)

	// Old WDT1 is not saved so nothing to restore.
}

func (a *Ast) ResetCpu() {
	// - Load 16 into WDT00 when reset/restart
	// 16 is a small value that will quickly trigger
	a.Mem().MustWrite32(WDT1_RLD_CTR, 16)
	a.Mem().MustWrite32(WDT1_RESTART, WDT_RESTART_PASSWORD)
	// 0x2 - Reset system after timeout
	// 0x1 - WDT enable
	a.Mem().MustWrite32(WDT1_CTRL, 0x3)
}
