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
	g           = (*gpioSystem)(nil)
)

type gpioSystem struct {
	f      *os.File
	button map[pb.Button]chan time.Duration
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

func (g *gpioSystem) manageButton(line string, b pb.Button) {
	// TODO(bluecmd): Assume the line is inverted for now, probably will
	// always be the case in all platforms though
	l := requestLineHandle(g.f, []uint32{linePortMap[line]}, []bool{true})
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

func startGpio(c string) {
	f, err := os.OpenFile(c, os.O_RDWR, 0600)
	if err != nil {
		log.Printf("startGpio: open: %v", err)
		return
	}

	g = &gpioSystem{
		f: f,
		button: map[pb.Button]chan time.Duration{
			pb.Button_BUTTON_POWER: make(chan time.Duration),
			pb.Button_BUTTON_RESET: make(chan time.Duration),
		},
	}

	// TODO(bluecmd): These are motherboard specific, figure out how
	// to have these configurable for other boards.
	go g.monitor([]string{
		"CPU0_FIVR_FAULT_N",
		"CPU0_PROCHOT_N",
		"CPU0_THERMTRIP_N",
		"CPU1_FIVR_FAULT_N",
		"CPU1_PROCHOT_N",
		"CPU1_THERMTRIP_N",
		"CPU_CATERR_N",
		"MB_SLOT_ID",
		"MEMAB_MEMHOT_N",
		"MEMCD_MEMHOT_N",
		"MEMEF_MEMHOT_N",
		"MEMGH_MEMHOT_N",
		"NMI_BTN_N",
		"PCH_BMC_THERMTRIP_N",
		"PCH_PWR_OK",
		"PWR_BTN_N",
		"RST_BTN_N",
		"SKU0",
		"SKU1",
		"SKU2",
		"SKU3",
		"SLP_S3_N",
		"SPI_SEL",
		"SYS_PWR_OK",
		"SYS_THROTTLE",
		"UART_SELECT0",
		"UART_SELECT1",
	})

	g.hog(map[string]bool{
		"BMC_NMI_N":      true,
		"BMC_SMI_INT_N":  true,
		"UNKN_E4":        true,
		"UNKN_PWR_CAP":   true,
		"BAT_SENSE_EN_N": false,
		"BIOS_SEL":       false,
		"FAST_PROCHOT":   false,
		"PWR_LED_N":      false,
		// TODO(bluecmd): Figure out what this controls
		"UNKN_Q4": false,
	})

	go g.manageButton("BMC_PWR_BTN_OUT_N", pb.Button_BUTTON_POWER)
	go g.manageButton("BMC_RST_BTN_OUT_N", pb.Button_BUTTON_RESET)
}
