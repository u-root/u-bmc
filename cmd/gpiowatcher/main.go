// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/u-root/u-bmc/pkg/ast2400"
	"github.com/u-root/u-bmc/pkg/platform"
)

var (
	// U4 and V2 is RMII receive clock probably never really interesting
	// V6 and O2 is the fan PWM input on the F06 Leopard
	// TODO(bluecmd): This can be made adaptive in the future
	ignoreLines = flag.String("ignore", "U4,V2,O0,O2", "Ignore events on the specified comma separated lines")
)

func main() {
	flag.Parse()

	a := ast2400.Open()
	defer a.Close()

	ignoredPorts := make(map[uint32]bool)
	for _, part := range strings.Split(*ignoreLines, ",") {
		ignoredPorts[ast2400.GpioPort(part)] = true
	}

	log.SetOutput(os.Stdout)

	// Dump multifunction pin setup
	fmt.Printf("SCU80: %08x\n", a.Mem().MustRead32(ast2400.SCU_BASE+0x80))
	fmt.Printf("SCU84: %08x\n", a.Mem().MustRead32(ast2400.SCU_BASE+0x84))
	fmt.Printf("SCU88: %08x\n", a.Mem().MustRead32(ast2400.SCU_BASE+0x88))
	fmt.Printf("SCU8C: %08x\n", a.Mem().MustRead32(ast2400.SCU_BASE+0x8C))
	fmt.Printf("SCU90: %08x\n", a.Mem().MustRead32(ast2400.SCU_BASE+0x90))
	fmt.Printf("SCU94: %08x\n", a.Mem().MustRead32(ast2400.SCU_BASE+0x94))

	_ = platform.LinePortMap()

	p := a.SnapshotGpio()
	dir := make(map[uint32]bool)
	for _, g := range p.List() {
		if g.State == ast2400.LINE_STATE_OUTPUT {
			dir[g.Port] = true
		}
	}

	for _, g := range p.List() {
		if g.State == ast2400.LINE_STATE_HIGH {
			log.Printf("%-30s high (output: %v)\n", portName(g.Port), dir[g.Port])
		} else if g.State == ast2400.LINE_STATE_LOW {
			log.Printf("%-30s low  (output: %v)\n", portName(g.Port), dir[g.Port])
		}
	}
	for {
		s := a.SnapshotGpio()

		if !p.Equal(s) {
			for _, g := range s.Diff(p) {
				if ignoredPorts[g.Port] {
					continue
				}
				if g.State == ast2400.LINE_STATE_BECAME_INPUT {
					log.Printf("%-30s became input\n", portName(g.Port))
				} else if g.State == ast2400.LINE_STATE_BECAME_OUTPUT {
					log.Printf("%-30s became output\n", portName(g.Port))
				} else if g.State == ast2400.LINE_STATE_BECAME_HIGH {
					log.Printf("%-30s became high\n", portName(g.Port))
				} else if g.State == ast2400.LINE_STATE_BECAME_LOW {
					log.Printf("%-30s became low\n", portName(g.Port))
				}
			}
		}
		p = s
		time.Sleep(10 * time.Millisecond)
	}
}


func portName(p uint32) string {
	n, ok := platform.GpioPortToName(p)
	if !ok {
		n = ast2400.GpioPortToFunction(p)
	}
	return n
}

