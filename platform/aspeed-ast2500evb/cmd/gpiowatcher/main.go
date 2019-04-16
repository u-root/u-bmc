// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"github.com/u-root/u-bmc/pkg/gpiowatcher"
	"github.com/u-root/u-bmc/platform/aspeed-ast2500evb/pkg/gpio"
)

var (
	doBinaryLog = flag.Bool("binary", false, "Record a binary log of all events instead of text")
	doPlayback  = flag.Bool("playback", false, "Play a binary log from stdin of events instead of capturing")
	// U4, T0, V2 is RMII and very noisy
	ignoreLines = flag.String("ignore", "U4,V2,T0", "Ignore events on the specified comma separated lines when printing")
)

func main() {
	flag.Parse()
	gpiowatcher.Gpiowatcher(*doBinaryLog, *doPlayback, *ignoreLines, gpiowatcher.NewAstPlatform(&gpio.Gpio{}))
}
