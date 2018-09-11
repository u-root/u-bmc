// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestDecode(t *testing.T) {
	e := gpioevent_data{}
	a := []byte{173, 111, 34, 156, 101, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0}
	f := bytes.NewBuffer(a)
	err := binary.Read(f, binary.LittleEndian, &e)
	if err != nil {
		t.Errorf("Failed at decoding: %v\n")
	}

	if e.Id != GPIOEVENT_EVENT_RISING_EDGE {
		t.Errorf("expected %d, got %d\n", GPIOEVENT_EVENT_RISING_EDGE, e.Id)
	}
}


