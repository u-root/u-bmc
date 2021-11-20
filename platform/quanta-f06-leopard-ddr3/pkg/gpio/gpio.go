// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpio

import (
	"github.com/u-root/u-bmc/pkg/hardware/aspeed"
)

var (
	linePortMap = map[string]uint32{
		"UNKN_BOOT1":          aspeed.GpioPort("A2"),
		"UNKN_PWR_CAP":        aspeed.GpioPort("A3"),
		"FAST_PROCHOT":        aspeed.GpioPort("B3"),
		"CPU0_THERMTRIP_N":    aspeed.GpioPort("B5"),
		"CPU1_THERMTRIP_N":    aspeed.GpioPort("B6"),
		"MEMAB_MEMHOT_N":      aspeed.GpioPort("C2"),
		"MEMCD_MEMHOT_N":      aspeed.GpioPort("C3"),
		"MEMEF_MEMHOT_N":      aspeed.GpioPort("C6"),
		"MEMGH_MEMHOT_N":      aspeed.GpioPort("C7"),
		"NMI_BTN_N":           aspeed.GpioPort("D0"),
		"BMC_NMI_N":           aspeed.GpioPort("D1"),
		"PWR_BTN_N":           aspeed.GpioPort("D2"),
		"BMC_PWR_BTN_OUT_N":   aspeed.GpioPort("D3"),
		"RST_BTN_N":           aspeed.GpioPort("D4"),
		"BMC_RST_BTN_OUT_N":   aspeed.GpioPort("D5"),
		"PCH_PWR_OK":          aspeed.GpioPort("E1"),
		"SYS_PWR_OK":          aspeed.GpioPort("E2"),
		"UNKN_E4":             aspeed.GpioPort("E4"),
		"BMC_SMI_INT_N":       aspeed.GpioPort("E5"),
		"PCH_BMC_THERMTRIP_N": aspeed.GpioPort("F0"),
		"CPU_CATERR_N":        aspeed.GpioPort("F1"),
		"SLP_S3_N":            aspeed.GpioPort("G2"),
		"UNKN_BOOT0":          aspeed.GpioPort("G3"),
		// TODO(bluecmd): This is what the Tioga Pass has, unverified
		"BAT_SENSE_EN_N": aspeed.GpioPort("G4"),
		"BIOS_SEL":       aspeed.GpioPort("N4"),
		// Tristate:
		// set to input to allow host to own BIOS flash
		// set to output to allow bmc to own BIOS flash
		"SPI_SEL":           aspeed.GpioPort("N5"),
		"UART_SELECT0":      aspeed.GpioPort("N6"),
		"UART_SELECT1":      aspeed.GpioPort("N7"),
		"SKU0":              aspeed.GpioPort("P0"),
		"SKU1":              aspeed.GpioPort("P1"),
		"SKU2":              aspeed.GpioPort("P2"),
		"SKU3":              aspeed.GpioPort("P3"),
		"CPU0_PROCHOT_N":    aspeed.GpioPort("P6"),
		"CPU1_PROCHOT_N":    aspeed.GpioPort("P7"),
		"UNKN_Q4":           aspeed.GpioPort("Q4"),
		"PWR_LED_N":         aspeed.GpioPort("Q5"),
		"CPU0_FIVR_FAULT_N": aspeed.GpioPort("Q6"),
		"CPU1_FIVR_FAULT_N": aspeed.GpioPort("Q7"),
		"MB_SLOT_ID":        aspeed.GpioPort("R1"),
		"SYS_THROTTLE":      aspeed.GpioPort("R4"),
	}

	// Reverse map of linePortMap
	portLineMap map[uint32]string
)

type Gpio struct {
}

func init() {
	portLineMap = make(map[uint32]string)
	for k, v := range linePortMap {
		portLineMap[v] = k
	}
}

func (_ *Gpio) GpioNameToPort(l string) (uint32, bool) {
	s, ok := linePortMap[l]
	return s, ok
}

func (_ *Gpio) GpioPortToName(i uint32) (string, bool) {
	s, ok := portLineMap[i]
	return s, ok
}
