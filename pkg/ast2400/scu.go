// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ast2400

const (
	// This is a static number that acts as a password to prevent
	// accidental memory writes that would screw up the system.
	// The SCU unlocks write access by writing this constant to the SCUs first
	// register. The SCU is locked for writes by writing any other value.
	// See AST2400 datasheet, SCU00: Protection Key Register
	SCU_PASSWORD uint32  = 0x1688A8A8
	SCU_BASE     uintptr = 0x1E6E2000

	SCU_DEFAULT_RESET uint32 = 0xFFCFFEDC
)

func (a *Ast) unlockScuWriteAccess() {
	// Unlock by writing password to SCU00
	a.Mem().MustWrite32(SCU_BASE+0, SCU_PASSWORD)
}

func (a *Ast) lockScuWriteAccess() {
	// Lock by writing anything other than the password to SCU00
	a.Mem().MustWrite32(SCU_BASE+0, 0x0)
}

func (a *Ast) GetHardwareStrapping() uint32 {
	// SCU70: Hardware Strapping Register
	return a.Mem().MustRead32(SCU_BASE + 0x70)
}

func (a *Ast) GetSiliconRevision() uint32 {
	// SCU7C: Silicon Revision Register
	return a.Mem().MustRead32(SCU_BASE + 0x7C)
}

func (a *Ast) ModelName() string {
	switch a.GetSiliconRevision() {
	case 0x00000102:
		return "AST2200-A0/A1"
	case 0x00000200:
		return "AST1100-A0 or AST2050-A0"
	case 0x00000201:
		return "AST1100-A1 or AST2050-A1"
	case 0x00000202:
		return "AST1100-A2/3 or AST2050-A2/3 or AST2150-A0/1"
	case 0x00000300:
		return "AST2100-A0"
	case 0x00000301:
		return "AST2100-A1"
	case 0x00000302:
		return "AST2100-A2/3"
	case 0x01000003:
		return "AST2300-A0"
	case 0x01010003:
		return "AST1300-A1"
	case 0x01010203:
		return "AST1050-A1"
	case 0x01010303:
		return "AST2300-A1"
	case 0x02000303:
		return "AST2400-A0"
	case 0x02010103:
		return "AST1400-A1"
	case 0x02010303:
		return "AST1250-A1 or AST2400-A1"
	}
	return ""
}

func (a *Ast) IsSpiMaster() bool {
	return a.GetHardwareStrapping()&(1<<12) > 0
}

func (a *Ast) SetSpiMaster(master bool) {
	a.unlockScuWriteAccess()
	defer a.lockScuWriteAccess()
	// Enable bit 12, SPI master
	v := a.GetHardwareStrapping() & ^uint32(1<<12)
	if master {
		v = v | (1 << 12)
	}
	// SCU70: Hardware Strapping Register
	a.Mem().MustWrite32(SCU_BASE+0x70, v)
}

func (a *Ast) setCpuEnable(en bool) {
	a.unlockScuWriteAccess()
	defer a.lockScuWriteAccess()
	v := a.GetHardwareStrapping() & ^uint32(3)
	if en {
		// Set boot from SPI flash memory
		v = v | 2
	} else {
		// Enable bit 0:1, Disable CPU operation
		v = v | 3
	}
	// SCU70: Hardware Strapping Register
	a.Mem().MustWrite32(SCU_BASE+0x70, v)
}

func (a *Ast) FreezeCpu() {
	a.setCpuEnable(false)
	a.DisableWdt()
}

func (a *Ast) UnfreezeCpu() {
	a.setCpuEnable(true)
	a.EnableWdt()
}

func (a *Ast) SetResetControl(v uint32) {
	a.unlockScuWriteAccess()
	defer a.lockScuWriteAccess()
	// SCU04: System Reset Control Register
	a.Mem().MustWrite32(SCU_BASE+0x4, v)
}
