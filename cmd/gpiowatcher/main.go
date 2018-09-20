// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/u-root/u-bmc/pkg/ast2400"
)

var (
	doBinaryLog = flag.Bool("binary", false, "Record a binary log of all events instead of text")
	doPlayback  = flag.Bool("playback", false, "Play a binary log from stdin of events instead of capturing")
)

type snapshoter interface {
	Close()
	SnapshotGpio() *ast2400.State
}

type outputer interface {
	Close()
	Log(s *ast2400.State)
}

func main() {
	flag.Parse()
	log.SetOutput(os.Stdout)

	var a snapshoter
	if *doPlayback {
		a = &playback{os.Stdin}
	} else {
		a = ast2400.Open()
	}
	defer a.Close()

	p := a.SnapshotGpio()

	var o outputer
	if *doBinaryLog {
		o = newBinaryLog(p)
	} else {
		o = newStdoutLog(p)
	}
	defer o.Close()

	for {
		s := a.SnapshotGpio()
		if s == nil {
			break
		}
		o.Log(s)
		// TOOD(bluecmd): When doing playback we should ideally load
		// the effective timestamp of every sample
		time.Sleep(10 * time.Millisecond)
	}
}
