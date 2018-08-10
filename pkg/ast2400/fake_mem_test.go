// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ast2400

import (
	"fmt"
	"testing"
)

type op struct {
	write   bool
	address uintptr
	data8   uint8
	data16  uint16
	data32  uint32
	size    int
}

type fakeMem struct {
	t   *testing.T
	ops []op
}

func opstr(o *op) string {
	t := "read"
	if o.write {
		t = "write"
	}
	d := 0
	if o.size == 8 {
		d = int(o.data8)
	}
	if o.size == 16 {
		d = int(o.data16)
	}
	if o.size == 32 {
		d = int(o.data32)
	}
	return fmt.Sprintf("{%s @ %08x, %v bit = %08x}", t, o.address, o.size, d)
}

func (m *fakeMem) MustRead32(a uintptr) uint32 {
	o := m.ops[0]
	m.ops = m.ops[1:]
	if o.write || o.address != a || o.size != 32 {
		m.t.Errorf("Expected %s, got 32 bit read on %08x", opstr(&o), a)
	}
	return o.data32
}

func (m *fakeMem) MustRead8(a uintptr) uint8 {
	o := m.ops[0]
	m.ops = m.ops[1:]
	if o.write || o.address != a || o.size != 8 {
		m.t.Errorf("Expected %s, got 8 bit read on %08x", opstr(&o), a)
	}
	return o.data8
}

func (m *fakeMem) MustWrite32(a uintptr, d uint32) {
	o := m.ops[0]
	m.ops = m.ops[1:]
	if !o.write || o.address != a || o.size != 32 || o.data32 != d {
		m.t.Errorf("Expected %s, got 32 bit write of %08x on %08x", opstr(&o), d, a)
	}
}

func (m *fakeMem) MustWrite8(a uintptr, d uint8) {
	o := m.ops[0]
	m.ops = m.ops[1:]
	if !o.write || o.address != a || o.size != 8 || o.data8 != d {
		m.t.Errorf("Expected %s, got 8 bit write of %02x on %08x", opstr(&o), d, a)
	}
}

func (m *fakeMem) ExpectWrite32(a uintptr, d uint32) {
	m.ops = append(m.ops, op{true, a, 0, 0, d, 32})
}

func (m *fakeMem) ExpectWrite8(a uintptr, d uint8) {
	m.ops = append(m.ops, op{true, a, d, 0, 0, 8})
}

func (m *fakeMem) FakeRead32(a uintptr, d uint32) {
	m.ops = append(m.ops, op{false, a, 0, 0, d, 32})
}

func (m *fakeMem) FakeRead8(a uintptr, d uint8) {
	m.ops = append(m.ops, op{false, a, d, 0, 0, 8})
}

func (m *fakeMem) Close() {
}

func fakeMemory(t *testing.T) *fakeMem {
	return &fakeMem{t, make([]op, 0)}
}
