// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpiowatcher

import (
	"log"
	"os"
	"time"

	"github.com/u-root/u-bmc/pkg/ast2400"
)

type GpioPlatform interface {
	GpioPortToName(p uint32) (string, bool)
}

type snapshoter interface {
	Close()
	SnapshotGpio() *ast2400.State
}

type outputer interface {
	Close()
	Log(s *ast2400.State)
}

type ast2400Platform struct {
	g GpioPlatform
}

func NewAst2400Platform(g GpioPlatform) *ast2400Platform {
	return &ast2400Platform{g}
}

func (a *ast2400Platform) PortName(p uint32) string {
	n, ok := a.g.GpioPortToName(p)
	if !ok {
		n = ast2400.GpioPortToFunction(p)
	}
	return n
}

// TODO(bluecmd): Things are still very ast2400 centric, but at least
// the interface to the platform implementions should be pretty stable.
type platform interface {
	PortName(p uint32) string
}

func Gpiowatcher(doBinaryLog bool, doPlayback bool, ignoreLines string, plt platform) {
	log.SetOutput(os.Stdout)

	var a snapshoter
	if doPlayback {
		a = &playback{os.Stdin}
	} else {
		a = ast2400.Open()
	}
	defer a.Close()

	p := a.SnapshotGpio()

	var o outputer
	if doBinaryLog {
		o = newBinaryLog(p)
	} else {
		o = newStdoutLog(p, ignoreLines, plt)
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
