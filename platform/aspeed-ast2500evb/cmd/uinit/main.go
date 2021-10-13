// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"github.com/u-root/u-bmc/pkg/bmc"
	"github.com/u-root/u-bmc/pkg/logger"
	"github.com/u-root/u-bmc/platform/aspeed-ast2500evb/pkg/platform"
)

var log = logger.LogContainer.GetSimpleLogger()

func main() {
	p := platform.Platform()
	defer p.Close()
	err, _ := bmc.Startup(p)
	if err != nil {
		log.Fatal(err)
	}
	for {
		bmc.Shell()
	}
}
