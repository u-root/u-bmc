// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpiowatcher

import (
	"log"
	"strings"

	"github.com/u-root/u-bmc/pkg/aspeed"
)

type stdoutLog struct {
	p      *aspeed.State
	dir    map[uint32]bool
	ignore map[uint32]bool
	plt    platform
}

func newStdoutLog(p *aspeed.State, ignoreLines string, plt platform) *stdoutLog {
	ignoredPorts := make(map[uint32]bool)
	for _, part := range strings.Split(ignoreLines, ",") {
		ignoredPorts[aspeed.GpioPort(part)] = true
	}

	l := stdoutLog{p, make(map[uint32]bool), ignoredPorts, plt}
	for _, g := range p.List() {
		if g.State == aspeed.LINE_STATE_OUTPUT {
			l.dir[g.Port] = true
		}
	}

	for _, g := range p.List() {
		if g.State == aspeed.LINE_STATE_HIGH {
			log.Printf("%-30s high (output: %v)\n", plt.PortName(g.Port), l.dir[g.Port])
		} else if g.State == aspeed.LINE_STATE_LOW {
			log.Printf("%-30s low  (output: %v)\n", plt.PortName(g.Port), l.dir[g.Port])
		} else if g.State == aspeed.LINE_STATE_SCU {
			log.Printf("SCU%02x is %08x (description: %s)\n", g.Port, p.Scu[g.Port], aspeed.ScuRegisterToFunction(g.Port))
		}
	}
	return &l
}

func (l *stdoutLog) Log(s *aspeed.State) {
	if !l.p.Equal(s) {
		for _, g := range s.Diff(l.p) {
			if l.ignore[g.Port] {
				continue
			}
			if g.State == aspeed.LINE_STATE_BECAME_INPUT {
				l.dir[g.Port] = false
				log.Printf("%-30s became input (value: %v)\n", l.plt.PortName(g.Port), s.PortValue(g.Port))
			} else if g.State == aspeed.LINE_STATE_BECAME_OUTPUT {
				l.dir[g.Port] = true
				log.Printf("%-30s became output (value: %v)\n", l.plt.PortName(g.Port), s.PortValue(g.Port))
			} else if g.State == aspeed.LINE_STATE_BECAME_HIGH {
				if l.dir[g.Port] {
					log.Printf("%-30s driving high\n", l.plt.PortName(g.Port))
				} else {
					log.Printf("%-30s sensing high\n", l.plt.PortName(g.Port))
				}
			} else if g.State == aspeed.LINE_STATE_BECAME_LOW {
				if l.dir[g.Port] {
					log.Printf("%-30s driving low\n", l.plt.PortName(g.Port))
				} else {
					log.Printf("%-30s sensing low\n", l.plt.PortName(g.Port))
				}
			} else if g.State == aspeed.LINE_STATE_SCU_CHANGED {
				log.Printf("SCU%02x is now %08x\n", g.Port, s.Scu[g.Port])
			}
		}
	}
	l.p = s
}

func (l *stdoutLog) Close() {
}
