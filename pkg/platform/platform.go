// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package platform

import (
	"github.com/u-root/u-bmc/pkg/ast2400"
)

var (
	linePortMap = map[string]uint32{
		"HAND_SW_ID1":             ast2400.GpioPort("R2"),
		"HAND_SW_ID2":             ast2400.GpioPort("R3"),
		"HAND_SW_ID3":             ast2400.GpioPort("R4"),
		"HAND_SW_ID4":             ast2400.GpioPort("R5"),
		"PWR_BTN_N":               ast2400.GpioPort("D0"),
		"RST_BTN_N":               ast2400.GpioPort("S0"),
		"HEARTBEAT_LED":           ast2400.GpioPort("Q7"),
		"USB_SW0":                 ast2400.GpioPort("E4"),
		"USB_SW1":                 ast2400.GpioPort("E5"),
		"USB_MUX_EN_N":            ast2400.GpioPort("S3"),
		"UART_SEL0":               ast2400.GpioPort("E0"),
		"UART_SEL1":               ast2400.GpioPort("E1"),
		"UART_SEL2":               ast2400.GpioPort("E2"),
		"UART_RX_EN":              ast2400.GpioPort("E3"),
		"POSTCODE_0":              ast2400.GpioPort("G0"),
		"POSTCODE_1":              ast2400.GpioPort("G1"),
		"POSTCODE_2":              ast2400.GpioPort("G2"),
		"POSTCODE_3":              ast2400.GpioPort("G3"),
		"POSTCODE_4":              ast2400.GpioPort("P4"),
		"POSTCODE_5":              ast2400.GpioPort("P5"),
		"POSTCODE_6":              ast2400.GpioPort("P6"),
		"POSTCODE_7":              ast2400.GpioPort("P7"),
		"DEBUG_CARD_PRESENT_N":    ast2400.GpioPort("R1"),
		"BMC_READY_N":             ast2400.GpioPort("D4"),
		"BMC_PWR_BTN_SLOT1_OUT_N": ast2400.GpioPort("D3"),
		"BMC_PWR_BTN_SLOT2_OUT_N": ast2400.GpioPort("D1"),
		"BMC_PWR_BTN_SLOT3_OUT_N": ast2400.GpioPort("D7"),
		"BMC_PWR_BTN_SLOT4_OUT_N": ast2400.GpioPort("D5"),
		"BMC_RST_BTN_SLOT1_OUT_N": ast2400.GpioPort("H1"),
		"BMC_RST_BTN_SLOT2_OUT_N": ast2400.GpioPort("H0"),
		"BMC_RST_BTN_SLOT3_OUT_N": ast2400.GpioPort("H3"),
		"BMC_RST_BTN_SLOT4_OUT_N": ast2400.GpioPort("H2"),
		"PWR_LED_SLOT1_N":         ast2400.GpioPort("M1"),
		"PWR_LED_SLOT2_N":         ast2400.GpioPort("M0"),
		"PWR_LED_SLOT3_N":         ast2400.GpioPort("M3"),
		"PWR_LED_SLOT4_N":         ast2400.GpioPort("M2"),
		"ID_LED_SLOT1_N":          ast2400.GpioPort("F1"),
		"ID_LED_SLOT2_N":          ast2400.GpioPort("F0"),
		"ID_LED_SLOT3_N":          ast2400.GpioPort("F3"),
		"ID_LED_SLOT4_N":          ast2400.GpioPort("F2"),
		"PRESENT_SLOT1_N":         ast2400.GpioPort("H5"),
		"PRESENT_SLOT2_N":         ast2400.GpioPort("H4"),
		"PRESENT_SLOT3_N":         ast2400.GpioPort("H7"),
		"PRESENT_SLOT4_N":         ast2400.GpioPort("H6"),
		"BIC_READY_SLOT1_N":       ast2400.GpioPort("N3"),
		"BIC_READY_SLOT2_N":       ast2400.GpioPort("N2"),
		"BIC_READY_SLOT3_N":       ast2400.GpioPort("N5"),
		"BIC_READY_SLOT4_N":       ast2400.GpioPort("N4"),
		"12V_EN_SLOT1_N":          ast2400.GpioPort("O5"),
		"12V_EN_SLOT2_N":          ast2400.GpioPort("O4"),
		"12V_EN_SLOT3_N":          ast2400.GpioPort("O7"),
		"12V_EN_SLOT4_N":          ast2400.GpioPort("O6"),
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
