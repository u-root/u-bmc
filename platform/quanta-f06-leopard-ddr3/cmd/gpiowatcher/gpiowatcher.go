// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"

	"github.com/u-root/u-bmc/pkg/hardware/gpiowatcher"
	"github.com/u-root/u-bmc/platform/quanta-f06-leopard-ddr3/pkg/gpio"
)

var (
	doBinaryLog = flag.Bool("binary", false, "Record a binary log of all events instead of text")
	doPlayback  = flag.Bool("playback", false, "Play a binary log from stdin of events instead of capturing")
	// U4 and V2 is RMII receive clock probably never really interesting
	// O0 and O2 is the fan tach input
	// N0 and N1 is the fan PWN output
	ignoreLines = flag.String("ignore", "U4,V2,O0,O2,N0,N1", "Ignore events on the specified comma separated lines when printing")
)

func main() {
	flag.Parse()
	gpiowatcher.Gpiowatcher(*doBinaryLog, *doPlayback, *ignoreLines, gpiowatcher.NewAstPlatform(&gpio.Gpio{}))
}
