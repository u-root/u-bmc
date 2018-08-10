// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"io"
	"log"
	"os"
	"syscall"

	"github.com/u-root/u-bmc/pkg/ast2400"
)

var (
	linePortMap = map[string]uint32{
		"UNKN_PWR_CAP":        ast2400.GpioPort("A3"),
		"FAST_PROCHOT":        ast2400.GpioPort("B3"),
		"CPU0_THERMTRIP_N":    ast2400.GpioPort("B5"),
		"CPU1_THERMTRIP_N":    ast2400.GpioPort("B6"),
		"MEMAB_MEMHOT_N":      ast2400.GpioPort("C2"),
		"MEMCD_MEMHOT_N":      ast2400.GpioPort("C3"),
		"MEMEF_MEMHOT_N":      ast2400.GpioPort("C6"),
		"MEMGH_MEMHOT_N":      ast2400.GpioPort("C7"),
		"NMI_BTN_N":           ast2400.GpioPort("D0"),
		"BMC_NMI_N":           ast2400.GpioPort("D1"),
		"PWR_BTN_N":           ast2400.GpioPort("D2"),
		"BMC_PWR_BTN_OUT_N":   ast2400.GpioPort("D3"),
		"RST_BTN_N":           ast2400.GpioPort("D4"),
		"BMC_RST_BTN_OUT_N":   ast2400.GpioPort("D5"),
		"PCH_PWR_OK":          ast2400.GpioPort("E1"),
		"SYS_PWR_OK":          ast2400.GpioPort("E2"),
		"UNKN_E4":             ast2400.GpioPort("E4"),
		"BMC_SMI_INT_N":       ast2400.GpioPort("E5"),
		"PCH_BMC_THERMTRIP_N": ast2400.GpioPort("F0"),
		// TODO(bluecmd): Verify
		"CPU_CATERR_N": ast2400.GpioPort("F1"),
		"SLP_S3_N":     ast2400.GpioPort("G2"),
		"BAT_DETECT":   ast2400.GpioPort("G4"),
		"UNKN_M4":      ast2400.GpioPort("M4"),
		"UNKN_M5":      ast2400.GpioPort("M5"),
		"UNKN_N2":      ast2400.GpioPort("N2"),
		"BIOS_SEL":     ast2400.GpioPort("N4"),
		// Tristate:
		// set to input to allow host to own BIOS flash
		// set to output to allow bmc to own BIOS flash
		"SPI_SEL":           ast2400.GpioPort("N5"),
		"UNKN_N6":           ast2400.GpioPort("N6"),
		"UNKN_N7":           ast2400.GpioPort("N7"),
		"SKU0":              ast2400.GpioPort("P0"),
		"SKU1":              ast2400.GpioPort("P1"),
		"SKU2":              ast2400.GpioPort("P2"),
		"SKU3":              ast2400.GpioPort("P3"),
		"UNKN_P4":           ast2400.GpioPort("P4"),
		"UNKN_P5":           ast2400.GpioPort("P5"),
		"CPU0_PROCHOT_N":    ast2400.GpioPort("P6"),
		"CPU1_PROCHOT_N":    ast2400.GpioPort("P7"),
		"UNKN_Q4":           ast2400.GpioPort("Q4"),
		"PWR_LED_N":         ast2400.GpioPort("Q5"),
		"CPU0_FIVR_FAULT_N": ast2400.GpioPort("Q6"),
		"CPU1_FIVR_FAULT_N": ast2400.GpioPort("Q7"),
		"MB_SLOT_ID":        ast2400.GpioPort("R1"),
		"SYS_THROTTLE":      ast2400.GpioPort("R4"),
	}
)

type gpioSystem struct {
	f *os.File
}

func (g *gpioSystem) monitorOne(line string) {
	e := getLineEvent(g.f, linePortMap[line], GPIOHANDLE_REQUEST_INPUT, GPIOEVENT_REQUEST_BOTH_EDGES)
	d := getLineValues(e)
	log.Printf("Monitoring GPIO line %-30s [initial value %v]", line, d.values[0])
	for {
		e := readEvent(e)
		if e == nil {
			break
		}
		log.Printf("%s: event %v", e)
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

func (g *gpioSystem) powerButton(line string) {
	// TODO(bluecmd): Assume the line is inverted for now, probably will
	// always be the case in all platforms though
	l := requestLineHandle(g.f, []uint32{linePortMap[line]}, []bool{true})
	log.Printf("Initialized power button %s", line)

	// TODO(bluecmd): replace with grpc
	syscall.Mknod("/tmp/power_btn", syscall.S_IFIFO|0600, 0)

	c := make([]byte, 1)
	for {
		cf, err := os.OpenFile("/tmp/power_btn", os.O_RDONLY, 0600)
		if err != nil {
			log.Printf("Power button FIFO failed: open: %v", err)
			break
		}
		_, err = cf.Read(c)
		if err == io.EOF {
			continue
		}
		if err != nil {
			log.Printf("Power button FIFO failed: read: %v", err)
			break
		}
		if c[0] == '1' {
			log.Printf("Pressing power button")
			setLineValues(l, []bool{false})
		}
		if c[0] == '0' {
			log.Printf("Releasing power button")
			setLineValues(l, []bool{true})
		}
		cf.Close()
	}
}

func startGpio(c string) {
	f, err := os.OpenFile(c, os.O_RDWR, 0600)
	if err != nil {
		log.Fatalf("startGpio: open: %v", err)
	}

	g := gpioSystem{f}

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
		"UNKN_M4",
		"UNKN_M5",
		"UNKN_N2",
		"UNKN_N6",
		"UNKN_N7",
		"UNKN_P4",
		"UNKN_P5",
	})

	g.hog(map[string]bool{
		"BMC_NMI_N":         true,
		"BMC_RST_BTN_OUT_N": true,
		"BMC_SMI_INT_N":     true,
		"UNKN_E4":           true,
		"UNKN_PWR_CAP":      true,
		"BAT_DETECT":        false,
		"BIOS_SEL":          false,
		"FAST_PROCHOT":      false,
		"PWR_LED_N":         false,
		// TODO(bluecmd): Figure out what this controls
		"UNKN_Q4": false,
	})

	go g.powerButton("BMC_PWR_BTN_OUT_N")
}
