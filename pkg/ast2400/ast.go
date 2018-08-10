// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Library for accessing AST2400 series BMC functions
//
// Usually packages like these contain a notice to say use on your own risk
// but this time, it's for real. During development of this library a lot of
// hard hungs have been had and power cycles have had to be made.
// Be warned.
//
// The library generally does not save original registers, so any modification
// will be in competition to the local OS on the BMC. Safest course of action
// is to call FreezeCpu(), do your thing, UnfreezeCpu() + ResetCpu(). That
// should avoid any complications.
//
// Call ast2400.Open() and ast2400.Close() as the first and last thing
// before and after you want to run any library commands.
//
// The library supports being run both on the host CPU and the BMC CPU.
// When not run on the BMC, it will use the LPC bus and the LPC2AHB feature
// of the SuperIO. If that has been disabled, running on the host CPU will
// not work.

package ast2400

import ()

type Ast struct {
	mem memProvider
}

func Open() *Ast {
	mem := openMem()
	a := &Ast{mem}

	if a.ModelName() == "" {
		panic("Could not detect supported SOC")
	}
	return a
}

func OpenWithMemory(mem memProvider) *Ast {
	return &Ast{mem}
}

func (a *Ast) Close() {
	a.mem.Close()
}
