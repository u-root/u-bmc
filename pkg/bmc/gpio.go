// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"fmt"
	"log"
	"os"
	"time"

	pb "github.com/u-root/u-bmc/proto"
)

type GpioPlatform interface {
	GpioNameToPort(string) (uint32, bool)
	GpioPortToName(uint32) (string, bool)
	InitializeGpio(g *GpioSystem) error
}

type GpioSystem struct {
	p      GpioPlatform
	f      *os.File
	button map[pb.Button]chan time.Duration
}

func (g *GpioSystem) monitorOne(line string) error {
	port, ok := g.p.GpioNameToPort(line)
	if !ok {
		return fmt.Errorf("Could not resolve GPIO %s", line)
	}
	e, err := getLineEvent(g.f, port, GPIOHANDLE_REQUEST_INPUT, GPIOEVENT_REQUEST_BOTH_EDGES)
	if err != nil {
		return err
	}
	d, err := getLineValues(e)
	if err != nil {
		return err
	}
	log.Printf("Monitoring GPIO line %-30s [initial value %v]", line, d.values[0])
	for {
		ev, err := readEvent(e)
		if err != nil {
			return err
		}
		if ev == nil {
			break
		}
		f := ""
		if ev.Id == GPIOEVENT_EVENT_FALLING_EDGE {
			f = "falling edge"
		} else if ev.Id == GPIOEVENT_EVENT_RISING_EDGE {
			f = "rising edge"
		} else {
			f = fmt.Sprintf("unknown event %v", ev)
		}
		log.Printf("%s: %s", line, f)

	}
	log.Printf("Monitoring stopped for GPIO line %s", line)
	return nil
}

func (g *GpioSystem) Monitor(lines []string) {
	for _, line := range lines {
		go func(l string) {
			err := g.monitorOne(l)
			if err != nil {
				log.Printf("Monitor %s failed: %v", l, err)
			}
		}(line)
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

	_, err := requestLineHandle(g.f, lidx, vals)
	if err != nil {
		log.Printf("Hog failed: %v", err)
	}
}

func (g *GpioSystem) PressButton(b pb.Button, durMs uint32) error {
	if durMs > 1000*10 {
		return fmt.Errorf("Maximum allowed depress duration is 10 seconds")
	}
	dur := time.Duration(durMs) * time.Millisecond
	c, ok := g.button[b]
	if !ok {
		return fmt.Errorf("Unknown button %v", b)
	}
	c <- dur
	return nil
}

func (g *GpioSystem) ManageButton(line string, b pb.Button) {
	// TODO(bluecmd): Assume the line is inverted for now, probably will
	// always be the case in all platforms though
	port, ok := g.p.GpioNameToPort(line)
	if !ok {
		log.Printf("Could not resolve GPIO %s", line)
		return
	}
	l, err := requestLineHandle(g.f, []uint32{port}, []bool{true})
	if err != nil {
		log.Printf("ManageButton %s failed: %v", line, err)
		return
	}
	log.Printf("Initialized button %s", line)

	for {
		dur := <-g.button[b]
		log.Printf("Pressing button %s", line)
		setLineValues(l, []bool{false})

		time.Sleep(dur)

		log.Printf("Releasing button %s", line)
		setLineValues(l, []bool{true})
	}
}

func startGpio(p GpioPlatform) (*GpioSystem, error) {
	f, err := os.OpenFile("/dev/gpiochip0", os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}

	g := GpioSystem{
		p: p,
		f: f,
		button: map[pb.Button]chan time.Duration{
			pb.Button_BUTTON_POWER: make(chan time.Duration),
			pb.Button_BUTTON_RESET: make(chan time.Duration),
		},
	}

	err = p.InitializeGpio(&g)
	if err != nil {
		return nil, fmt.Errorf("platform.InitializeGpio: %v", err)
	}
	return &g, nil
}
