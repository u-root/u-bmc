// Copyright 2018-2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aspeed

import (
	"testing"
)

func TestFreezeCpu(t *testing.T) {
	fm := fakeMemory(t)
	a := OpenWithMemory(fm)
	fm.ExpectWrite32(0x1E6E2000, SCU_PASSWORD)
	fm.FakeRead32(0x1E6E2070, 0x0)
	fm.ExpectWrite32(0x1E6E2070, 0x3)
	fm.ExpectWrite32(0x1E6E2000, 0)
	// DisableWdt
	fm.ExpectWrite32(0x1e78500c, 0)
	fm.ExpectWrite32(0x1e78502c, 0)
	a.FreezeCpu()
}

func TestUnfreezeCpu(t *testing.T) {
	fm := fakeMemory(t)
	a := OpenWithMemory(fm)
	fm.ExpectWrite32(0x1E6E2000, SCU_PASSWORD)
	fm.FakeRead32(0x1E6E2070, 0xffffffff)
	fm.ExpectWrite32(0x1E6E2070, 0xfffffffe)
	fm.ExpectWrite32(0x1E6E2000, 0)
	// EnableWdt
	fm.ExpectWrite32(0x1e785034, 1)
	fm.ExpectWrite32(0x1e78502c, 0x82)
	a.UnfreezeCpu()
}

func TestSpiMaster(t *testing.T) {
	fm := fakeMemory(t)
	a := OpenWithMemory(fm)
	fm.ExpectWrite32(0x1E6E2000, SCU_PASSWORD)
	fm.FakeRead32(0x1E6E2070, 0x0)
	fm.ExpectWrite32(0x1E6E2070, 1<<12)
	fm.ExpectWrite32(0x1E6E2000, 0)
	a.SetSpiMaster(true)
	fm.ExpectWrite32(0x1E6E2000, SCU_PASSWORD)
	fm.FakeRead32(0x1E6E2070, 0xffffffff)
	fm.ExpectWrite32(0x1E6E2070, 0xffffefff)
	fm.ExpectWrite32(0x1E6E2000, 0)
	a.SetSpiMaster(false)
	fm.FakeRead32(0x1E6E2070, 0x1000)
	if !a.IsSpiMaster() {
		t.Errorf("Expected SPI master, was not\n")
	}
	fm.FakeRead32(0x1E6E2070, 0)
	if a.IsSpiMaster() {
		t.Errorf("Expected not SPI master, was\n")
	}
}
