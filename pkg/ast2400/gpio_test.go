// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ast2400

import (
	"testing"
)

func TestGpioPort(t *testing.T) {
	if GpioPort("D1") != 0x19 {
		t.Errorf("Port D1 did not resolve to 0x19\n")
	}
	if GpioPort("r4") != 0x8c {
		t.Errorf("Port R4 did not resolve to 0x8c\n")
	}
}
