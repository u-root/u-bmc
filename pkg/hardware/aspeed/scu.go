// Copyright 2018-2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aspeed

import (
	"fmt"
)

var (
	scuRegs = map[uint32]string{
		0x00:  "Protection Key Register",
		0x04:  "System Reset Control Register",
		0x08:  "Clock Selection Register",
		0x0C:  "Clock Stop Control Register",
		0x10:  "Frequency Counter Control Register",
		0x14:  "Frequency Counter Measurement Register",
		0x18:  "Interrupt Control and Status Register",
		0x1C:  "D2-PLL Parameter Register",
		0x20:  "M-PLL Parameter Register",
		0x24:  "H-PLL Parameter Register",
		0x28:  "Frequency counter comparison range",
		0x2C:  "Misc. Control Register",
		0x30:  "PCI Configuration Setting Register #1",
		0x34:  "PCI Configuration Setting Register #2",
		0x38:  "PCI Configuration Setting Register #3",
		0x3C:  "System Reset Control/Status Register",
		0x40:  "SOC Scratch Register #1",
		0x44:  "SOC Scratch Register #2",
		0x48:  "MAC Interface Clock Delay Setting",
		0x4C:  "Misc. 2 Control Register",
		0x50:  "VGA Scratch Register #1",
		0x54:  "VGA Scratch Register #2",
		0x58:  "VGA Scratch Register #3",
		0x5C:  "VGA Scratch Register #4",
		0x60:  "VGA Scratch Register #5",
		0x64:  "VGA Scratch Register #6",
		0x68:  "VGA Scratch Register #7",
		0x6C:  "VGA Scratch Register #8",
		0x70:  "Hardware Strapping Register",
		0x74:  "Random Number Generator Control",
		0x78:  "Random Number Generator Data Output",
		0x7C:  "Silicon Revision ID Register",
		0x80:  "Multi-function Pin Control #1",
		0x84:  "Multi-function Pin Control #2",
		0x88:  "Multi-function Pin Control #3",
		0x8C:  "Multi-function Pin Control #4",
		0x90:  "Multi-function Pin Control #5",
		0x94:  "Multi-function Pin Control #6",
		0x9C:  "Watchdog Reset Selection",
		0xA0:  "Multi-function Pin Control #7",
		0xA4:  "Multi-function Pin Control #8",
		0xA8:  "Multi-function Pin Control #9",
		0xC0:  "Power Saving Wakeup Enable Register",
		0xC4:  "Power Saving Wakeup Control Register",
		0xD0:  "Hardware Strapping Register Set 2",
		0xE0:  "SCU Free Run Counter Read Back #4",
		0xE4:  "SCU Free Run Counter Extended Read Back #4",
		0x100: "Coprocessor (CPU2) Control Register",
	}
)

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

func (a *Ast) ModelName() (string, error) {
	names := map[uint32]string{
		0x00000102: "AST2200-A0/A1",
		0x00000200: "AST1100-A0 or AST2050-A0",
		0x00000201: "AST1100-A1 or AST2050-A1",
		0x00000202: "AST1100-A2/3 or AST2050-A2/3 or AST2150-A0/1",
		0x00000300: "AST2100-A0",
		0x00000301: "AST2100-A1",
		0x00000302: "AST2100-A2/3",
		0x01000003: "AST2300-A0",
		0x01010003: "AST1300-A1",
		0x01010203: "AST1050-A1",
		0x01010303: "AST2300-A1",
		0x02000303: "AST2400-A0",
		0x02010103: "AST1400-A1",
		0x02010303: "AST1250-A1 or AST2400-A1",
		0x04000303: "AST2500-A0",
		0x04000103: "AST2510-A0",
		0x04000203: "AST2520-A0",
		0x04000403: "AST2530-A0",
		0x04010303: "AST2500-A1",
		0x04010103: "AST2510-A1",
		0x04010203: "AST2520-A1",
		0x04010403: "AST2530-A1",
		0x04030303: "AST2500-A2",
		0x04030103: "AST2510-A2",
		0x04030203: "AST2520-A2",
		0x04030403: "AST2530-A2",
	}
	rev := a.GetSiliconRevision()
	if name, ok := names[rev]; ok {
		return name, nil
	}
	return "", fmt.Errorf("unknown revision %#08x", rev)
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

func ScuRegisterToFunction(r uint32) string {
	return scuRegs[r]
}
