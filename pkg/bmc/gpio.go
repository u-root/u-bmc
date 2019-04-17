// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	pb "github.com/u-root/u-bmc/proto"
)

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
	SetValues(out []bool) error
	Close()
}

type gpioEventImpl interface {
	WaitForEvent() (int, error)
	State() (bool, error)
}

type gpioImpl interface {
	RequestLineHandle(lines []uint32, out []bool) (gpioLineImpl, error)
	GetLineEvent(line uint32) (gpioEventImpl, error)
}

type unknownEventError struct {
	Err error
}

type GpioSystem struct {
	p      GpioPlatform
	impl   gpioImpl
	Button map[pb.Button]chan chan bool
}

type GpioCallback func(line string, c chan bool, initial bool)

var (
	gpioLine = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "ubmc",
		Subsystem: "gpio",
		Name:      "line",
		Help:      "Monitored u-bmc GPIO line",
	}, []string{"line"})
)

func IsUnknownGpioEvent(err error) bool {
	switch err.(type) {
	case *unknownEventError:
		return true
	}
	return false
}

func (e *unknownEventError) Error() string {
	return e.Err.Error()
}

func init() {
	prometheus.MustRegister(gpioLine)
}

func LogGpio(line string, c chan bool, d bool) {
	log.Printf("Monitoring GPIO line %-30s [initial value %v]", line, d)
	m := gpioLine.With(prometheus.Labels{"line": line})
	if d {
		m.Set(1)
	}
	for value := range c {
		f := ""
		if value {
			m.Set(1)
			f = "rising edge"
		} else {
			m.Set(0)
			f = "falling edge"
		}
		log.Printf("%s: %s", line, f)
	}
}

func (g *GpioSystem) monitorOne(line string, cb GpioCallback) error {
	port, ok := g.p.GpioNameToPort(line)
	if !ok {
		return fmt.Errorf("Could not resolve GPIO %s", line)
	}
	e, err := g.impl.GetLineEvent(port)
	if err != nil {
		return err
	}
	d, err := e.State()
	if err != nil {
		return err
	}

	c := make(chan bool)
	go cb(line, c, d)

	for {
		ev, err := e.WaitForEvent()
		if err != nil && !IsUnknownGpioEvent(err) {
			close(c)
			return err
		}

		switch ev {
		case GPIO_EVENT_FALLING_EDGE:
			c <- false
		case GPIO_EVENT_RISING_EDGE:
			c <- true
		default:
			log.Printf("Received unknown event on GPIO line %s: %v", line, err)
		}
	}
	// Should be unreachable
}

func (g *GpioSystem) Monitor(lines map[string]GpioCallback) {
	log.Printf("Setting up %v GPIO monitors", len(lines))
	for line, cb := range lines {
		// TODO(bluecmd): This is a bit redundant, but there have been cases
		// where u-bmc starts up but no monitors are started. This logging statement
		// is here to help pin-point the issue if it happens again. If there are no
		// reports of that happening, this log line can be removed.
		log.Printf("Starting monitor for GPIO %v", line)
		go func(l string, cb GpioCallback) {
			err := g.monitorOne(l, cb)
			if err != nil {
				log.Printf("Monitor %s failed: %v", l, err)
			}
		}(line, cb)
	}
}

func (g *GpioSystem) Hog(lines map[string]bool) {
	// TODO(bluecmd): There is a hard limit of 64 lines per kernel handle,
	// if we ever hit that we will have to change this part.
	if len(lines) > 64 {
		log.Printf("Too many GPIO lines to hog: %d > 64", len(lines))
		return
	}
	lidx := make([]uint32, len(lines))
	vals := make([]bool, len(lines))
	i := 0
	for l, v := range lines {
		port, ok := g.p.GpioNameToPort(l)
		if !ok {
			log.Printf("Could not resolve GPIO %s", l)
			return
		}
		lidx[i] = port
		vals[i] = v
		log.Printf("Hogging GPIO line %-30s = %v", l, v)
		i++
	}

	_, err := g.impl.RequestLineHandle(lidx, vals)
	if err != nil {
		log.Printf("Hog failed: %v", err)
	}
	// Don't close the line handle, keep it opened as long as we're alive to
	// really hog the line
}

func (g *GpioSystem) PressButton(ctx context.Context, b pb.Button, durMs uint32) (chan bool, error) {
	if durMs > 1000*10 {
		return nil, fmt.Errorf("Maximum allowed depress duration is 10 seconds")
	}
	dur := time.Duration(durMs) * time.Millisecond
	c, ok := g.Button[b]
	if !ok {
		return nil, fmt.Errorf("Unknown button %v", b)
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

func (g *GpioSystem) ManageHeartbeat(line string, dur time.Duration) {
	port, ok := g.p.GpioNameToPort(line)
	if !ok {
		log.Printf("Could not resolve GPIO %s", line)
		return
	}
	l, err := g.impl.RequestLineHandle([]uint32{port}, []bool{true})
	if err != nil {
		log.Printf("ManageButton %s failed: %v", line, err)
		return
	}
	defer l.Close()
	hdur := time.Duration(dur.Nanoseconds()/2) * time.Nanosecond
	log.Printf("Initialized heartbeat %s", line)
	for {
		l.SetValues([]bool{true})
		time.Sleep(hdur)
		l.SetValues([]bool{false})
		time.Sleep(hdur)
	}
}

func (g *GpioSystem) ManageButton(line string, b pb.Button, flags int) {
	port, ok := g.p.GpioNameToPort(line)
	if !ok {
		log.Printf("Could not resolve GPIO %s", line)
		return
	}
	l, err := g.impl.RequestLineHandle([]uint32{port}, []bool{true})
	if err != nil {
		log.Printf("ManageButton %s failed: %v", line, err)
		return
	}
	defer l.Close()
	log.Printf("Initialized button %s", line)

	for {
		pushc := <-g.Button[b]

		for p := range pushc {
			if p {
				log.Printf("Pressing button %s", line)
			} else {
				log.Printf("Releasing button %s", line)
			}
			if flags&GPIO_INVERTED != 0 {
				p = !p
			}
			l.SetValues([]bool{p})
		}
	}
}

func (g *GpioSystem) ManageOutput(line string, c chan bool) {
	port, ok := g.p.GpioNameToPort(line)
	if !ok {
		log.Printf("Could not resolve GPIO %s", line)
		return
	}
	l, err := g.impl.RequestLineHandle([]uint32{port}, []bool{true})
	if err != nil {
		log.Printf("SetLine %s failed: %v", line, err)
		return
	}
	defer l.Close()

	for {
		l.SetValues([]bool{<-c})
	}
}

func startGpio(p GpioPlatform) (*GpioSystem, error) {
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
		p:    p,
		impl: impl,
		Button: map[pb.Button]chan chan bool{
			pb.Button_BUTTON_POWER: make(chan chan bool),
			pb.Button_BUTTON_RESET: make(chan chan bool),
		},
	}
	return &g
}
