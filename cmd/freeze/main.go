// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/u-root/u-bmc/pkg/ast2400"
)

func main() {
	a := ast2400.Open()
	defer a.Close()

	a.FreezeCpu()
}
