// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
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
		} else if g.State == ast2400.LINE_STATE_SCU {
			log.Printf("SCU%02x is %08x (description: %s)\n", g.Port, p.Scu(g.Port), ast2400.ScuRegisterToFunction(g.Port))
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
					dir[g.Port] = false
					log.Printf("%-30s became input (value: %v)\n", portName(g.Port), s.PortValue(g.Port))
				} else if g.State == ast2400.LINE_STATE_BECAME_OUTPUT {
					dir[g.Port] = true
					log.Printf("%-30s became output (value: %v)\n", portName(g.Port), s.PortValue(g.Port))
				} else if g.State == ast2400.LINE_STATE_BECAME_HIGH {
					if dir[g.Port] {
						log.Printf("%-30s driving high\n", portName(g.Port))
					} else {
						log.Printf("%-30s sensing high\n", portName(g.Port))
					}
				} else if g.State == ast2400.LINE_STATE_BECAME_LOW {
					if dir[g.Port] {
						log.Printf("%-30s driving low\n", portName(g.Port))
					} else {
						log.Printf("%-30s sensing low\n", portName(g.Port))
					}
				} else if g.State == ast2400.LINE_STATE_SCU_CHANGED {
					log.Printf("SCU%02x is now %08x\n", g.Port, s.Scu(g.Port))
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
