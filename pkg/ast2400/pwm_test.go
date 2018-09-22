// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ast2400

import (
	"testing"
)

func TestPwmFanDiv4(t *testing.T) {
	fm := fakeMemory(t)
	a := OpenWithMemory(fm)
	fm.ExpectWrite32(0x1e786028, 0)
	fm.ExpectWrite32(0x1e786028, 1<<0)
	fm.FakeRead32(0x1e786010, 0x10000001)
	fm.FakeRead32(0x1e78602c, 0x80015e87)
	rpm := a.MeasureFanRpm(0)
	expected := 2005
	if rpm != expected {
		t.Errorf("Fan RPM calculation failed, expected %v got %v", expected, rpm)
	}
}

func TestPwmFanDoubleEdgeDiv4(t *testing.T) {
	fm := fakeMemory(t)
	a := OpenWithMemory(fm)
	fm.ExpectWrite32(0x1e786028, 0)
	fm.ExpectWrite32(0x1e786028, 1<<0)
	fm.FakeRead32(0x1e786010, 0x02100021)
	fm.FakeRead32(0x1e78602c, 0x80015e87)
	rpm := a.MeasureFanRpm(0)
	expected := 1002
	if rpm != expected {
		t.Errorf("Fan RPM calculation failed, expected %v got %v", expected, rpm)
	}
}
