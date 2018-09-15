// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/u-root/u-bmc/pkg/platform"
	pb "github.com/u-root/u-bmc/proto"
)

var (
	linePortMap = platform.LinePortMap()
	g = (*gpioSystem)(nil)
)

type gpioSystem struct {
	f           *os.File
	powerButton chan time.Duration
}

func (g *gpioSystem) monitorOne(line string) {
	e := getLineEvent(g.f, linePortMap[line], GPIOHANDLE_REQUEST_INPUT, GPIOEVENT_REQUEST_BOTH_EDGES)
	d := getLineValues(e)
	log.Printf("Monitoring GPIO line %-30s [initial value %v]", line, d.values[0])
	for {
		ev := readEvent(e)
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
		// TODO(bluecmd): Just to be sure, read the value (there is a race but
		// I just to double check that the edge detection works somewhat well)
		log.Printf("%s: %s, value is now %d", line, f, getLineValues(e).values[0])

	}
	log.Printf("Monitoring stopped for GPIO line %s", line)
}

func (g *gpioSystem) monitor(lines []string) {
	for _, line := range lines {
		go g.monitorOne(line)
	}
}

func (g *gpioSystem) hog(lines map[string]bool) {
	// TODO(bluecmd): There is a hard limit of 64 lines per kernel handle,
	// if we ever hit that we will have to change this part.
	if len(lines) > 64 {
		panic("Too many GPIO lines to hog")
	}
	lidx := make([]uint32, len(lines))
	vals := make([]bool, len(lines))
	i := 0
	for l, v := range lines {
		lidx[i] = linePortMap[l]
		vals[i] = v
		log.Printf("Hogging GPIO line %-30s = %v", l, v)
		i++
	}

	requestLineHandle(g.f, lidx, vals)
}

func PressButton(b pb.Button, durMs uint32) error {
	if durMs > 1000 * 10 {
		return fmt.Errorf("Maximum allowed depress duration is 10 seconds")
	}
	if b == pb.Button_POWER {
		g.powerButton <- time.Duration(durMs) * time.Millisecond
	} else {
		return fmt.Errorf("Unknown button %v", b)
	}
	return nil
}

func (g *gpioSystem) managePowerButton(line string) {
	// TODO(bluecmd): Assume the line is inverted for now, probably will
	// always be the case in all platforms though
	l := requestLineHandle(g.f, []uint32{linePortMap[line]}, []bool{true})
	log.Printf("Initialized power button %s", line)

	for {
		dur := <-g.powerButton
		log.Printf("Pressing power button")
		setLineValues(l, []bool{false})

		time.Sleep(dur)

		log.Printf("Releasing power button")
		setLineValues(l, []bool{true})
	}
}

func (g *gpioSystem) manageClock(line string, p time.Duration) {
	l := requestLineHandle(g.f, []uint32{linePortMap[line]}, []bool{false})
	log.Printf("Initialized clock %s", line)

	for {
		setLineValues(l, []bool{false})
		time.Sleep(p / 2)
		setLineValues(l, []bool{true})
		time.Sleep(p / 2)
	}
}

func startGpio(c string) {
	f, err := os.OpenFile(c, os.O_RDWR, 0600)
	if err != nil {
		log.Fatalf("startGpio: open: %v", err)
	}

	g = &gpioSystem{
		f: f,
		powerButton: make(chan time.Duration),
	}

	// TODO(bluecmd): These are motherboard specific, figure out how
	// to have these configurable for other boards.
	g.monitor([]string{
		"PWR_BTN_N",
		"RST_BTN_N",
		"HAND_SW_ID1",
		"HAND_SW_ID2",
		"HAND_SW_ID3",
		"HAND_SW_ID4",
		"DEBUG_CARD_PRESENT_N",
		"PRESENT_SLOT1_N",
		"PRESENT_SLOT2_N",
		"PRESENT_SLOT3_N",
		"PRESENT_SLOT4_N",
		"BIC_READY_SLOT1_N",
		"BIC_READY_SLOT2_N",
		"BIC_READY_SLOT3_N",
		"BIC_READY_SLOT4_N",
	})

	g.hog(map[string]bool{
		"BMC_READY_N":             true,
		"BMC_PWR_BTN_SLOT2_OUT_N": true,
		"BMC_PWR_BTN_SLOT3_OUT_N": true,
		"BMC_PWR_BTN_SLOT4_OUT_N": true,
		"BMC_RST_BTN_SLOT1_OUT_N": true,
		"BMC_RST_BTN_SLOT2_OUT_N": true,
		"BMC_RST_BTN_SLOT3_OUT_N": true,
		"BMC_RST_BTN_SLOT4_OUT_N": true,
		"12V_EN_SLOT1": true,
		"12V_EN_SLOT2": false,
		"12V_EN_SLOT3": false,
		"12V_EN_SLOT4": false,
		"POSTCODE_0": true,
		"POSTCODE_1": true,
		"POSTCODE_2": true,
		"POSTCODE_3": true,
		"POSTCODE_4": true,
		"POSTCODE_5": true,
		"POSTCODE_6": true,
		"POSTCODE_7": true,
	})

	go g.managePowerButton("BMC_PWR_BTN_SLOT1_OUT_N")
	go g.manageClock("ID_LED_SLOT1_N", time.Millisecond * time.Duration(500))
	go g.manageClock("ID_LED_SLOT2_N", time.Millisecond * time.Duration(1000))
	go g.manageClock("ID_LED_SLOT3_N", time.Millisecond * time.Duration(1500))
	go g.manageClock("ID_LED_SLOT4_N", time.Millisecond * time.Duration(2000))

	go g.manageClock("PWR_LED_SLOT1_N", time.Millisecond * time.Duration(500))
	go g.manageClock("PWR_LED_SLOT2_N", time.Millisecond * time.Duration(1000))
	go g.manageClock("PWR_LED_SLOT3_N", time.Millisecond * time.Duration(1500))
	go g.manageClock("PWR_LED_SLOT4_N", time.Millisecond * time.Duration(2000))

	go g.manageClock("HEARTBEAT_LED", time.Millisecond * time.Duration(750))
}
