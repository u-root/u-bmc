// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/u-root/u-bmc/pkg/bmc"
	"github.com/u-root/u-bmc/platform/aspeed-ast2500evb/pkg/platform"
)

func main() {
	p := platform.Platform()
	defer p.Close()
	bmc.Startup(p)
	for {
		bmc.Shell()
	}
}
