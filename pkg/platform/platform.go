// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package platform

import (
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
		"CPU_CATERR_N":        ast2400.GpioPort("F1"),
		"SLP_S3_N":            ast2400.GpioPort("G2"),
		"BAT_DETECT":          ast2400.GpioPort("G4"),
		"UNKN_M4":             ast2400.GpioPort("M4"),
		"UNKN_M5":             ast2400.GpioPort("M5"),
		"UNKN_N2":             ast2400.GpioPort("N2"),
		"BIOS_SEL":            ast2400.GpioPort("N4"),
		// Tristate:
		// set to input to allow host to own BIOS flash
		// set to output to allow bmc to own BIOS flash
		"SPI_SEL":             ast2400.GpioPort("N5"),
		"UNKN_N6":             ast2400.GpioPort("N6"),
		"UNKN_N7":             ast2400.GpioPort("N7"),
		"SKU0":                ast2400.GpioPort("P0"),
		"SKU1":                ast2400.GpioPort("P1"),
		"SKU2":                ast2400.GpioPort("P2"),
		"SKU3":                ast2400.GpioPort("P3"),
		"UNKN_P4":             ast2400.GpioPort("P4"),
		"UNKN_P5":             ast2400.GpioPort("P5"),
		"CPU0_PROCHOT_N":      ast2400.GpioPort("P6"),
		"CPU1_PROCHOT_N":      ast2400.GpioPort("P7"),
		"UNKN_Q4":             ast2400.GpioPort("Q4"),
		"PWR_LED_N":           ast2400.GpioPort("Q5"),
		"CPU0_FIVR_FAULT_N"  : ast2400.GpioPort("Q6"),
		"CPU1_FIVR_FAULT_N"  : ast2400.GpioPort("Q7"),
		"MB_SLOT_ID":          ast2400.GpioPort("R1"),
		"SYS_THROTTLE":        ast2400.GpioPort("R4"),
		"CPLD_CLK":            ast2400.GpioPort("B0"),
	}

	// Reverse map of linePortMap
	portLineMap map[uint32]string
)

func init() {
	portLineMap = make(map[uint32]string)
	for k, v := range linePortMap {
		portLineMap[v] = k
	}
}

func LinePortMap() map[string]uint32 {
	// TODO(bluecmd): This will need to be abstracted away somehow if more
	// platforms are to be supported.
	return linePortMap
}

func GpioPortToName(p uint32) (string, bool) {
	s, ok := portLineMap[p]
	return s, ok
}
