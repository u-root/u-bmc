// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ast2400

import (
	"fmt"
)

const (
	PWM_BASE uintptr = 0x1e786000
)

func (a *Ast) DumpPwm() {
	fmt.Printf(" PTCR00: General Control Register              %08x\n", a.Mem().MustRead32(PWM_BASE+0x0))
	fmt.Printf(" PTCR04: Clock Control Register                %08x\n", a.Mem().MustRead32(PWM_BASE+0x4))
	fmt.Printf(" PTCR08: Duty Control 0 Register               %08x\n", a.Mem().MustRead32(PWM_BASE+0x8))
	fmt.Printf(" PTCR0C: Duty Control 1 Register               %08x\n", a.Mem().MustRead32(PWM_BASE+0xc))
	fmt.Printf(" PTCR10: Type M Control 0 Register             %08x\n", a.Mem().MustRead32(PWM_BASE+0x10))
	fmt.Printf(" PTCR14: Type M Control 1 Register             %08x\n", a.Mem().MustRead32(PWM_BASE+0x14))
	fmt.Printf(" PTCR18: Type N Control 0 Register             %08x\n", a.Mem().MustRead32(PWM_BASE+0x18))
	fmt.Printf(" PTCR1C: Type N Control 1 Register             %08x\n", a.Mem().MustRead32(PWM_BASE+0x1c))
	fmt.Printf(" PTCR20: Tach Source Register                  %08x\n", a.Mem().MustRead32(PWM_BASE+0x20))
	fmt.Printf(" PTCR28: Trigger Register                      %08x\n", a.Mem().MustRead32(PWM_BASE+0x28))
	fmt.Printf(" PTCR2C: Result Register                       %08x\n", a.Mem().MustRead32(PWM_BASE+0x2c))
	fmt.Printf(" PTCR30: Interrupt Control Register            %08x\n", a.Mem().MustRead32(PWM_BASE+0x30))
	fmt.Printf(" PTCR34: Interrupt Status Register             %08x\n", a.Mem().MustRead32(PWM_BASE+0x34))
	fmt.Printf(" PTCR38: Type M Limit Register                 %08x\n", a.Mem().MustRead32(PWM_BASE+0x38))
	fmt.Printf(" PTCR3C: Type N Limit Register                 %08x\n", a.Mem().MustRead32(PWM_BASE+0x3C))
	fmt.Printf(" PTCR40: General Control Extension #1 Register %08x\n", a.Mem().MustRead32(PWM_BASE+0x40))
	fmt.Printf(" PTCR44: Clock Control Extension #1 Register   %08x\n", a.Mem().MustRead32(PWM_BASE+0x44))
	fmt.Printf(" PTCR48: Duty Control 2 Register               %08x\n", a.Mem().MustRead32(PWM_BASE+0x48))
	fmt.Printf(" PTCR4C: Duty Control 3 Register               %08x\n", a.Mem().MustRead32(PWM_BASE+0x4C))
	fmt.Printf(" PTCR50: Type O Control 0 Register             %08x\n", a.Mem().MustRead32(PWM_BASE+0x50))
	fmt.Printf(" PTCR54: Type O Control 1 Register             %08x\n", a.Mem().MustRead32(PWM_BASE+0x54))
	fmt.Printf(" PTCR60: Tach Source Extension #1 Register     %08x\n", a.Mem().MustRead32(PWM_BASE+0x60))
	fmt.Printf(" PTCR78: Type O Limit Register                 %08x\n", a.Mem().MustRead32(PWM_BASE+0x78))
}

func (a *Ast) MeasureFanRpm(fan uint) int {
	// PTCR28: Trigger Register
	// Reads are triggered by 0->1 transitions
	a.Mem().MustWrite32(PWM_BASE+0x28, 0)
	a.Mem().MustWrite32(PWM_BASE+0x28, 1<<fan)

	// Read tach divider and clock mode from PTCR10
	ctrl := a.Mem().MustRead32(PWM_BASE+0x10)
	div := 4 << (((ctrl >> 1) & 0x7) * 2)
	both := (ctrl & 0x20) != 0
	if both {
		div = div * 2
	}
	v := uint32(0)
	for v&(uint32(1)<<31) == 0 {
		// Wait for the measurement to be taken
		// PTCR2C: Result Register
		v = a.Mem().MustRead32(PWM_BASE + 0x2c)
	}
	v = v & ^(uint32(1) << 31)
	return (24 * 1000 * 1000 * 60) / (2 * int(v) * int(div))
}

func (a *Ast) SetFanDutyCycle(fan uint, p uint8) {
	v := a.Mem().MustRead32(PWM_BASE + 0x8)
	if fan >= 2 {
		panic("Fan must be 0 < x < 2")
	}
	// PTCR08: Duty Control 0 Register
	if fan == 0 {
		v = (v & 0xffff00ff) | uint32(p)<<8
	} else if fan == 1 {
		v = (v & 0x00ffffff) | uint32(p)<<24
	}
	a.Mem().MustWrite32(PWM_BASE+0x8, v)
}
