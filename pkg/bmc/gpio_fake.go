// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"sync"
)

type fakeGpio struct {
	p     GpioPlatform
	ports map[uint32]chan bool
	v     map[uint32]bool
	lock  *sync.Mutex
}

type fakeGpioEvent struct {
	g    *fakeGpio
	line uint32
}

type fakeGpioLine struct {
	g     *fakeGpio
	lines []uint32
}

func (l *fakeGpioLine) setValues(vals []bool) error {
	l.g.lock.Lock()
	defer l.g.lock.Unlock()
	for i, v := range vals {
		p := l.lines[i]
		l.g.v[p] = v
		pn, _ := l.g.p.GpioPortToName(p)
		log.Infof("FakeGpio: System set port %v to %v\n", pn, v)
		select {
		case l.g.ports[p] <- v:
		default:
		}
	}
	return nil
}

func (g *fakeGpio) Set(port uint32, v bool) {
	g.lock.Lock()
	c := g.ports[port]
	pn, _ := g.p.GpioPortToName(port)
	log.Infof("FakeGpio: Test harness set port %v to %v\n", pn, v)
	g.v[port] = v
	g.lock.Unlock()
	c <- v
}

func (g *fakeGpio) Current(port uint32) bool {
	g.lock.Lock()
	defer g.lock.Unlock()
	return g.v[port]
}

func (g *fakeGpio) WaitForChange(port uint32) bool {
	g.lock.Lock()
	c := g.ports[port]
	g.lock.Unlock()
	return <-c
}

func (g *fakeGpio) requestLineHandle(lines []uint32, out []bool) (gpioLineImpl, error) {
	return &fakeGpioLine{g, lines}, nil
}

func (g *fakeGpio) getLineEvent(line uint32) (gpioEventImpl, error) {
	return &fakeGpioEvent{g, line}, nil
}

func (e *fakeGpioEvent) getValue() (bool, error) {
	e.g.lock.Lock()
	defer e.g.lock.Unlock()
	return e.g.v[e.line], nil
}

func (e *fakeGpioEvent) read() (*int, error) {
	e.g.lock.Lock()
	c := e.g.ports[e.line]
	e.g.lock.Unlock()
	nv := <-c
	v := 0
	if nv {
		v = GPIO_EVENT_RISING_EDGE
	} else {
		v = GPIO_EVENT_FALLING_EDGE
	}
	return &v, nil
}

func FakeGpioImpl(p GpioPlatform, startupState map[uint32]bool) *fakeGpio {
	g := &fakeGpio{p, make(map[uint32]chan bool), make(map[uint32]bool), &sync.Mutex{}}
	for p, v := range startupState {
		g.ports[p] = make(chan bool)
		g.v[p] = v
	}
	return g
}
