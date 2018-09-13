// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ast2400

import (
	"reflect"
	"testing"
)

func TestGpioPort(t *testing.T) {
	if GpioPort("D1") != 0x19 {
		t.Errorf("Port D1 did not resolve to 0x19")
	}
	if GpioPort("r4") != 0x8c {
		t.Errorf("Port R4 did not resolve to 0x8c")
	}
	if GpioPortToName(0x8c) != "R4" {
		t.Errorf("0x8c did not resolve to R4")
	}
	if GpioPortToName(0xd9) != "AB1" {
		t.Errorf("0xd9 did not resolve to AB1")
	}
	if GpioPortToFunction(0xd9) != "ROMA19/GPOAB1/VPOR1" {
		t.Errorf("0xd9 did not resolve to function ROMA19/GPOAB1/VPOR1")
	}
}

func TestGpioBecomeHighChange(t *testing.T) {
	s1 := &state{make(map[uint32]uint32)}
	s2 := &state{make(map[uint32]uint32)}

	// Configure port I3 as output with low output
	s1.r[0x070] = 0
	s1.r[0x074] = 0x8

	// .. that changes to high
	s2.r[0x070] = 0x8
	s2.r[0x074] = 0x8

	if s1.Equal(s2) {
		t.Fatalf("s1 should not equal s2n")
	}

	d := s2.Diff(s1)
	expected := []lineState{{GpioPort("I3"), LINE_STATE_BECAME_HIGH}}

	if !reflect.DeepEqual(d, expected) {
		t.Errorf("Diff is not as expected, is %v, expected %v", d, expected)
	}
}

func TestGpioBecomeOutputChange(t *testing.T) {
	s1 := &state{make(map[uint32]uint32)}
	s2 := &state{make(map[uint32]uint32)}

	// Configure port I3 as input with high input
	s1.r[0x070] = 0x8
	s1.r[0x074] = 0

	// .. that changes to high output
	s2.r[0x070] = 0x8
	s2.r[0x074] = 0x8

	if s1.Equal(s2) {
		t.Fatalf("s1 should not equal s2n")
	}

	d := s2.Diff(s1)
	expected := []lineState{{GpioPort("I3"), LINE_STATE_BECAME_OUTPUT}}

	if !reflect.DeepEqual(d, expected) {
		t.Errorf("Diff is not as expected, is %v, expected %v", d, expected)
	}
}
