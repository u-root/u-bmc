// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpio

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/u-root/u-bmc/pkg/service/grpc/proto"
	"github.com/u-root/u-bmc/pkg/service/logger"
	"github.com/u-root/u-bmc/pkg/service/metric"
)

var log = logger.LogContainer.GetSimpleLogger()

const (
	GPIO_INVERTED = 0x1

	GPIO_EVENT_UNKNOWN      = 0
	GPIO_EVENT_RISING_EDGE  = 1
	GPIO_EVENT_FALLING_EDGE = 2
)

type GpioPlatform interface {
	GpioNameToPort(string) (uint32, bool)
	GpioPortToName(uint32) (string, bool)
	InitializeGpio(g *GpioSystem) error
}

type gpioLineImpl interface {
	setValues(out []bool) error
}

type gpioEventImpl interface {
	read() (*int, error)
	getValue() (bool, error)
}

type gpioImpl interface {
	requestLineHandle(lines []uint32, out []bool) (gpioLineImpl, error)
	getLineEvent(line uint32) (gpioEventImpl, error)
}

type GpioSystem struct {
	p      GpioPlatform
	impl   gpioImpl
	button map[proto.Button]chan chan bool
	m      sync.RWMutex
}

type GpioCallback func(line string, c chan bool, initial bool)

func LogGpio(line string, c chan bool, d bool) {
	log.Infof("Monitoring GPIO line %-30s [initial value %v]", line, d)
	var edge string
	metric.Gauge(metric.MetricOpts{
		Namespace: "ubmc",
		Subsystem: "gpio",
		Name:      "line",
	}, []string{`line="` + line + `"`}, func() float64 {
		if d {
			return 1
		}
		for value := range c {
			if value {
				edge = "rising edge"
				return 1
			} else {
				edge = "falling edge"
				return 0
			}

		}
		return -1
	})
	log.Infof("%s: %s", line, edge)
}

func (g *GpioSystem) monitorOne(line string, cb GpioCallback) error {
	port, ok := g.p.GpioNameToPort(line)
	if !ok {
		return fmt.Errorf("could not resolve GPIO %s", line)
	}
	e, err := g.impl.getLineEvent(port)
	if err != nil {
		return err
	}
	d, err := e.getValue()
	if err != nil {
		return err
	}

	c := make(chan bool)
	go cb(line, c, d)

	for {
		ev, err := e.read()
		if ev == nil && err != nil {
			return err
		}
		if ev == nil {
			break
		}

		switch *ev {
		case GPIO_EVENT_FALLING_EDGE:
			c <- false
		case GPIO_EVENT_RISING_EDGE:
			c <- true
		default:
			log.Errorf("Received unknown event on GPIO line %s: %v", line, err)
		}
	}
	close(c)
	log.Infof("Monitoring stopped for GPIO line %s", line)
	return nil
}

func (g *GpioSystem) Monitor(lines map[string]GpioCallback) {
	log.Infof("Setting up %v GPIO monitors", len(lines))
	for line, cb := range lines {
		// TODO(bluecmd): This is a bit redundant, but there have been cases
		// where u-bmc starts up but no monitors are started. This logging statement
		// is here to help pin-point the issue if it happens again. If there are no
		// reports of that happening, this log line can be removed.
		log.Infof("Starting monitor for GPIO %v", line)
		go func(l string, cb GpioCallback) {
			err := g.monitorOne(l, cb)
			if err != nil {
				log.Errorf("Monitor %s failed: %v", l, err)
			}
		}(line, cb)
	}
}

func (g *GpioSystem) Hog(lines map[string]bool) {
	// TODO(bluecmd): There is a hard limit of 64 lines per kernel handle,
	// if we ever hit that we will have to change this part.
	if len(lines) > 64 {
		log.Errorf("Too many GPIO lines to hog: %d > 64", len(lines))
		return
	}
	lidx := make([]uint32, len(lines))
	vals := make([]bool, len(lines))
	i := 0
	for l, v := range lines {
		port, ok := g.p.GpioNameToPort(l)
		if !ok {
			log.Errorf("Could not resolve GPIO %s", l)
			return
		}
		lidx[i] = port
		vals[i] = v
		log.Infof("Hogging GPIO line %-30s = %v", l, v)
		i++
	}

	_, err := g.impl.requestLineHandle(lidx, vals)
	if err != nil {
		log.Errorf("Hog failed: %v", err)
	}
}

func (g *GpioSystem) Button(b proto.Button) chan chan bool {
	g.m.Lock()
	defer g.m.Unlock()
	_, found := g.button[b]
	if !found {
		g.button[b] = make(chan chan bool)
	}
	return g.button[b]
}

func (g *GpioSystem) PressButton(ctx context.Context, b proto.Button, durMs uint32) (chan bool, error) {
	if durMs > 1000*10 {
		return nil, fmt.Errorf("maximum allowed depress duration is 10 seconds")
	}
	dur := time.Duration(durMs) * time.Millisecond
	g.m.RLock()
	defer g.m.RUnlock()
	c, ok := g.button[b]
	if !ok {
		return nil, fmt.Errorf("unknown button %v", b)
	}

	cc := make(chan bool)
	pushc := make(chan bool)
	// Queue the push behind any other push currently in action
	c <- pushc
	go func() {
		// Ensure the push has not been cancelled before starting
		if ctx.Err() != nil {
			close(pushc)
			return
		}
		// Commit to the push, and signal completion best-effort
		// as the caller might have gone away by the time the push is done
		pushc <- true
		time.Sleep(dur)
		pushc <- false
		close(pushc)
		select {
		case cc <- true:
		default:
		}
	}()
	return cc, nil
}

func (g *GpioSystem) ManageButton(line string, b proto.Button, flags int) {
	port, ok := g.p.GpioNameToPort(line)
	if !ok {
		log.Errorf("Could not resolve GPIO %s", line)
		return
	}
	l, err := g.impl.requestLineHandle([]uint32{port}, []bool{true})
	if err != nil {
		log.Errorf("ManageButton %s failed: %v", line, err)
		return
	}
	c := g.Button(b)
	log.Infof("Initialized button %s", line)

	for {
		pushc := <-c

		for p := range pushc {
			if p {
				log.Infof("Pressing button %s", line)
			} else {
				log.Infof("Releasing button %s", line)
			}
			if flags&GPIO_INVERTED != 0 {
				p = !p
			}
			err = l.setValues([]bool{p})
			if err != nil {
				log.Error(err)
			}
		}
	}
}

func StartGpio(p GpioPlatform) (*GpioSystem, error) {
	f, err := os.OpenFile("/dev/gpiochip0", os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	g := NewGpioSystem(p, &gpioLnx{f})

	err = p.InitializeGpio(g)
	if err != nil {
		return nil, fmt.Errorf("platform.InitializeGpio: %v", err)
	}
	return g, nil
}

func NewGpioSystem(p GpioPlatform, impl gpioImpl) *GpioSystem {
	g := GpioSystem{
		p:      p,
		impl:   impl,
		button: map[proto.Button]chan chan bool{},
	}
	return &g
}
