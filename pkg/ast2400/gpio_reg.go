// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ast2400

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

type gpio struct {
	pin  string
	name string
}

type gpioRegSet struct {
	set  string
	pins int
}

type gpioReg struct {
	sets []gpioRegSet
}

type LineState struct {
	Port  uint32
	State int
}

var (
	gpios = map[uint32]gpio{
		GpioPort("A0"):  {"D6", "GPIOA0/MAC1LINK"},
		GpioPort("A1"):  {"B5", "GPIOA1/MAC2LINK"},
		GpioPort("A2"):  {"A4", "GPIOA2/TIMER3"},
		GpioPort("A3"):  {"E6", "GPIOA3/TIMER4"},
		GpioPort("A4"):  {"C5", "GPIOA4/TIMER5/SCL9"},
		GpioPort("A5"):  {"B4", "GPIOA5/TIMER6/SDA9"},
		GpioPort("A6"):  {"A3", "GPIOA6/TIMER7/MDC2"},
		GpioPort("A7"):  {"D5", "GPIOA7/TIMER8/MDIO2"},
		GpioPort("B0"):  {"J21", "GPIOB0/SALT1"},
		GpioPort("B1"):  {"J20", "GPIOB1/SALT2"},
		GpioPort("B2"):  {"H18", "GPIOB2/SALT3"},
		GpioPort("B3"):  {"F18", "GPIOB3/SALT4"},
		GpioPort("B4"):  {"E19", "GPIOB4/LPCRST#"},
		GpioPort("B5"):  {"H19", "GPIOB5/LPCPD#/LPCSMI#"},
		GpioPort("B6"):  {"H20", "GPIOB6/LPCPME#"},
		GpioPort("B7"):  {"E18", "GPIOB7/EXTRST#/SPICS1#"},
		GpioPort("C0"):  {"C4", "GPIOC0/SD1CLK/SCL10"},
		GpioPort("C1"):  {"B3", "GPIOC1/SD1CMD/SDA10"},
		GpioPort("C2"):  {"A2", "GPIOC2/SD1DAT0/SCL11"},
		GpioPort("C3"):  {"E5", "GPIOC3/SD1DAT1/SDA11"},
		GpioPort("C4"):  {"D4", "GPIOC4/SD1DAT2/SCL12"},
		GpioPort("C5"):  {"C3", "GPIOC5/SD1DAT3/SDA12"},
		GpioPort("C6"):  {"B2", "GPIOC6/SD1CD#/SCL13"},
		GpioPort("C7"):  {"A1", "GPIOC7/SD1WP#/SDA13"},
		GpioPort("D0"):  {"A18", "GPIOD0/SD2CLK"},
		GpioPort("D1"):  {"D16", "GPIOD1/SD2CMD"},
		GpioPort("D2"):  {"B17", "GPIOD2/SD2DAT0"},
		GpioPort("D3"):  {"A17", "GPIOD3/SD2DAT1"},
		GpioPort("D4"):  {"C16", "GPIOD4/SD2DAT2"},
		GpioPort("D5"):  {"B16", "GPIOD5/SD2DAT3"},
		GpioPort("D6"):  {"A16", "GPIOD6/SD2CD#"},
		GpioPort("D7"):  {"E15", "GPIOD7/SD2WP#"},
		GpioPort("E0"):  {"D15", "GPIOE0/NCTS3"},
		GpioPort("E1"):  {"C15", "GPIOE1/NDCD3"},
		GpioPort("E2"):  {"B15", "GPIOE2/NDSR3"},
		GpioPort("E3"):  {"A15", "GPIOE3/NRI3"},
		GpioPort("E4"):  {"E14", "GPIOE4/NDTR3"},
		GpioPort("E5"):  {"D14", "GPIOE5/NRTS3"},
		GpioPort("E6"):  {"C14", "GPIOE6/TXD3"},
		GpioPort("E7"):  {"B14", "GPIOE7/RXD3"},
		GpioPort("F0"):  {"D18", "GPIOF0/NCTS4"},
		GpioPort("F1"):  {"B19", "GPIOF1/NDCD4/SIOPBI#"},
		GpioPort("F2"):  {"A20", "GPIOF2/NDSR4/SIOPWRGD"},
		GpioPort("F3"):  {"D17", "GPIOF3/NRI4/SIOPBO#"},
		GpioPort("F4"):  {"B18", "GPIOF4/NDTR4"},
		GpioPort("F5"):  {"A19", "GPIOF5/NRTS4/SIOSCI#"},
		GpioPort("F6"):  {"E16", "GPIOF6/TXD4"},
		GpioPort("F7"):  {"C17", "GPIOF7/RXD4"},
		GpioPort("G0"):  {"A14", "GPIOG0/SGPSCK"},
		GpioPort("G1"):  {"E13", "GPIOG1/SGPSLD"},
		GpioPort("G2"):  {"D13", "GPIOG2/SGPSI0"},
		GpioPort("G3"):  {"C13", "GPIOG3/SGPSI1"},
		GpioPort("G4"):  {"B13", "GPIOG4/WDTRST1/OSCCLK"},
		GpioPort("G5"):  {"Y21", "GPIOG5/WDTRST2/USBCKI"},
		GpioPort("G6"):  {"AA22", "GPIOG6/FLBUSY#"},
		GpioPort("G7"):  {"U18", "GPIOG7/FLWP#"},
		GpioPort("H0"):  {"A8", "GPIOH0/ROMD8/NCTS6"},
		GpioPort("H1"):  {"C7", "GPIOH1/ROMD9/NDCD6"},
		GpioPort("H2"):  {"B7", "GPIOH2/ROMD10/NDSR6"},
		GpioPort("H3"):  {"A7", "GPIOH3/ROMD11/NRI6"},
		GpioPort("H4"):  {"D7", "GPIOH4/ROMD12/NDTR6"},
		GpioPort("H5"):  {"B6", "GPIOH5/ROMD13/NRTS6"},
		GpioPort("H6"):  {"A6", "GPIOH6/ROMD14/TXD6"},
		GpioPort("H7"):  {"E7", "GPIOH7/ROMD15/RXD6"},
		GpioPort("I0"):  {"C22", "GPIOI0/SYSCS#"},
		GpioPort("I1"):  {"G18", "GPIOI1/SYSCK"},
		GpioPort("I2"):  {"D19", "GPIOI2/SYSDO"},
		GpioPort("I3"):  {"C20", "GPIOI3/SYSDI"},
		GpioPort("I4"):  {"B22", "GPIOI4/SPICS#/VBCS#"},
		GpioPort("I5"):  {"G19", "GPIOI5/SPICK/VBCK"},
		GpioPort("I6"):  {"C18", "GPIOI6/SPIDO/VBDO"},
		GpioPort("I7"):  {"E20", "GPIOI7/SPIDI/VBDI"},
		GpioPort("J0"):  {"J5", "GPIOJ0/SGPMCK"},
		GpioPort("J1"):  {"J4", "GPIOJ1/SGPMLD"},
		GpioPort("J2"):  {"K5", "GPIOJ2/SGPMO"},
		GpioPort("J3"):  {"J3", "GPIOJ3/SGPMI"},
		GpioPort("J4"):  {"T4", "VGAHS/GPIOJ4"},
		GpioPort("J5"):  {"U2", "VGAVS/GPIOJ5"},
		GpioPort("J6"):  {"T2", "DDCCLK/GPIOJ6"},
		GpioPort("J7"):  {"T1", "DDCDAT/GPIOJ7"},
		GpioPort("K0"):  {"E3", "GPIOK0/SCL5"},
		GpioPort("K1"):  {"D2", "GPIOK1/SDA5"},
		GpioPort("K2"):  {"C1", "GPIOK2/SCL6"},
		GpioPort("K3"):  {"F4", "GPIOK3/SDA6"},
		GpioPort("K4"):  {"E2", "GPIOK4/SCL7"},
		GpioPort("K5"):  {"D1", "GPIOK5/SDA7"},
		GpioPort("K6"):  {"G5", "GPIOK6/SCL8"},
		GpioPort("K7"):  {"F3", "GPIOK7/SDA8"},
		GpioPort("L0"):  {"U1", "GPIOL0/NCTS1"},
		GpioPort("L1"):  {"T5", "GPIOL1/NDCD1/VPIDE"},
		GpioPort("L2"):  {"U3", "GPIOL2/NDSR1/VPIODD"},
		GpioPort("L3"):  {"V1", "GPIOL3/NRI1/VPIHS"},
		GpioPort("L4"):  {"U4", "GPIOL4/NDTR1/VPIVS"},
		GpioPort("L5"):  {"V2", "GPIOL5/NRTS1/VPICLK"},
		GpioPort("L6"):  {"W1", "GPIOL6/TXD1/VPIB0"},
		GpioPort("L7"):  {"U5", "GPIOL7/RXD1/VPIB1"},
		GpioPort("M0"):  {"V3", "GPIOM0/NCTS2/VPIB2"},
		GpioPort("M1"):  {"W2", "GPIOM1/NDCD2/VPIB3"},
		GpioPort("M2"):  {"Y1", "GPIOM2/NDSR2/VPIB4"},
		GpioPort("M3"):  {"V4", "GPIOM3/NRI2/VPIB5"},
		GpioPort("M4"):  {"W3", "GPIOM4/NDTR2/VPIB6"},
		GpioPort("M5"):  {"Y2", "GPIOM5/NRTS2/VPIB7"},
		GpioPort("M6"):  {"AA1", "GPIOM6/TXD2/VPIB8"},
		GpioPort("M7"):  {"V5", "GPIOM7/RXD2/VPIB9"},
		GpioPort("N0"):  {"W4", "GPION0/PWM0/VPIG0"},
		GpioPort("N1"):  {"Y3", "GPION1/PWM1/VPIG1"},
		GpioPort("N2"):  {"AA2", "GPION2/PWM2/VPIG2"},
		GpioPort("N3"):  {"AB1", "GPION3/PWM3/VPIG3"},
		GpioPort("N4"):  {"W5", "GPION4/PWM4/VPIG4"},
		GpioPort("N5"):  {"Y4", "GPION5/PWM5/VPIG5"},
		GpioPort("N6"):  {"AA3", "GPION6/PWM6/VPIG6"},
		GpioPort("N7"):  {"AB2", "GPION7/PWM7/VPIG7"},
		GpioPort("O0"):  {"V6", "GPIOO0/TACH0/VPIG8"},
		GpioPort("O1"):  {"Y5", "GPIOO1/TACH1/VPIG9"},
		GpioPort("O2"):  {"AA4", "GPIOO2/TACH2/VPIR0"},
		GpioPort("O3"):  {"AB3", "GPIOO3/TACH3/VPIR1"},
		GpioPort("O4"):  {"W6", "GPIOO4/TACH4/VPIR2"},
		GpioPort("O5"):  {"AA5", "GPIOO5/TACH5/VPIR3"},
		GpioPort("O6"):  {"AB4", "GPIOO6/TACH6/VPIR4"},
		GpioPort("O7"):  {"V7", "GPIOO7/TACH7/VPIR5"},
		GpioPort("P0"):  {"Y6", "GPIOP0/TACH8/VPIR6"},
		GpioPort("P1"):  {"AB5", "GPIOP1/TACH9/VPIR7"},
		GpioPort("P2"):  {"W7", "GPIOP2/TACH10/VPIR8"},
		GpioPort("P3"):  {"AA6", "GPIOP3/TACH11/VPIR9"},
		GpioPort("P4"):  {"AB6", "GPIOP4/TACH12"},
		GpioPort("P5"):  {"Y7", "GPIOP5/TACH13"},
		GpioPort("P6"):  {"AA7", "GPIOP6/TACH14/BMCINT"},
		GpioPort("P7"):  {"AB7", "GPIOP7/TACH15/FLACK"},
		GpioPort("Q0"):  {"D3", "GPIOQ0/SCL3"},
		GpioPort("Q1"):  {"C2", "GPIOQ1/SDA3"},
		GpioPort("Q2"):  {"B1", "GPIOQ2/SCL4"},
		GpioPort("Q3"):  {"F5", "GPIOQ3/SDA4"},
		GpioPort("Q4"):  {"H4", "GPIOQ4/SCL14"},
		GpioPort("Q5"):  {"H3", "GPIOQ5/SDA14"},
		GpioPort("Q6"):  {"H2", "GPIOQ6"},
		GpioPort("Q7"):  {"H1", "GPIOQ7"},
		GpioPort("R0"):  {"V20", "GPIOR0/ROMCS1#"},
		GpioPort("R1"):  {"W21", "GPIOR1/ROMCS2#"},
		GpioPort("R2"):  {"Y22", "GPIOR2/ROMCS3#"},
		GpioPort("R3"):  {"U19", "GPIOR3/ROMCS4#"},
		GpioPort("R4"):  {"V21", "GPIOR4/ROMA24/VPOR6"},
		GpioPort("R5"):  {"W22", "GPIOR5/ROMA25/VPOR7"},
		GpioPort("R6"):  {"C6", "GPIOR6/MDC1"},
		GpioPort("R7"):  {"A5", "GPIOR7/MDIO1"},
		GpioPort("S0"):  {"U21", "ROMD4/GPIOS0/VPODE"},
		GpioPort("S1"):  {"T19", "ROMD5/GPIOS1/VPOHS"},
		GpioPort("S2"):  {"V22", "ROMD6/GPIOS2/VPOVS"},
		GpioPort("S3"):  {"U20", "ROMD7/GPIOS3/VPOCLK"},
		GpioPort("S4"):  {"R18", "ROMOE#/GPIOS4"},
		GpioPort("S5"):  {"N21", "ROMWE#/GPIOS5"},
		GpioPort("S6"):  {"L22", "ROMA22/GPIOS6/VPOR4"},
		GpioPort("S7"):  {"K18", "ROMA23/GPIOS7/VPOR5"},
		GpioPort("T0"):  {"A12", "RGMII1TXCK/RMII1TXEN/GPIOT0"},
		GpioPort("T1"):  {"B12", "RGMII1TXCTL/GPIOT1"},
		GpioPort("T2"):  {"C12", "RGMII1TXD0/RMII1TXD0/GPIOT2"},
		GpioPort("T3"):  {"D12", "RGMII1TXD1/RMII1TXD1/GPIOT3"},
		GpioPort("T4"):  {"E12", "RGMII1TXD2/GPIOT4"},
		GpioPort("T5"):  {"A13", "RGMII1TXD3/GPIOT5"},
		GpioPort("T6"):  {"D9", "RGMII2TXCK/RMII2TXEN/GPIOT6"},
		GpioPort("T7"):  {"E9", "RGMII2TXCTL/GPIOT7"},
		GpioPort("U0"):  {"A10", "RGMII2TXD0/RMII2TXD0/GPIOU0"},
		GpioPort("U1"):  {"B10", "RGMII2TXD1/RMII2TXD1/GPIOU1"},
		GpioPort("U2"):  {"C10", "RGMII2TXD2/GPIOU2"},
		GpioPort("U3"):  {"D10", "RGMII2TXD3/GPIOU3"},
		GpioPort("U4"):  {"E11", "RGMII1RXCK/RMII1RCLK/GPIOU4"},
		GpioPort("U5"):  {"D11", "RGMII1RXCTL/GPIOU5"},
		GpioPort("U6"):  {"C11", "RGMII1RXD0/RMII1RXD0/GPIOU6"},
		GpioPort("U7"):  {"B11", "RGMII1RXD1/RMII1RXD1/GPIOU7"},
		GpioPort("V0"):  {"A11", "RGMII1RXD2/RMII1CRSDV/GPIOV0"},
		GpioPort("V1"):  {"E10", "RGMII1RXD3/RMII1RXER/GPIOV1"},
		GpioPort("V2"):  {"C9", "RGMII2RXCK/RMII2RCLK/GPIOV2"},
		GpioPort("V3"):  {"B9", "RGMII2RXCTL/GPIOV3"},
		GpioPort("V4"):  {"A9", "RGMII2RXD0/RMII2RXD0/GPIOV4"},
		GpioPort("V5"):  {"E8", "RGMII2RXD1/RMII2RXD1/GPIOV5"},
		GpioPort("V6"):  {"D8", "RGMII2RXD2/RMII2CRSDV/GPIOV6"},
		GpioPort("V7"):  {"C8", "RGMII2RXD3/RMII2RXER/GPIOV7"},
		GpioPort("W0"):  {"L5", "ADC0/GPIW0"},
		GpioPort("W1"):  {"L4", "ADC1/GPIW1"},
		GpioPort("W2"):  {"L3", "ADC2/GPIW2"},
		GpioPort("W3"):  {"L2", "ADC3/GPIW3"},
		GpioPort("W4"):  {"L1", "ADC4/GPIW4"},
		GpioPort("W5"):  {"M5", "ADC5/GPIW5"},
		GpioPort("W6"):  {"M4", "ADC6/GPIW6"},
		GpioPort("W7"):  {"M3", "ADC7/GPIW7"},
		GpioPort("X0"):  {"M2", "ADC8/GPIX0"},
		GpioPort("X1"):  {"M1", "ADC9/GPIX1"},
		GpioPort("X2"):  {"N5", "ADC10/GPIX2"},
		GpioPort("X3"):  {"N4", "ADC11/GPIX3"},
		GpioPort("X4"):  {"N3", "ADC12/GPIX4"},
		GpioPort("X5"):  {"N2", "ADC13/GPIX5"},
		GpioPort("X6"):  {"N1", "ADC14/GPIX6"},
		GpioPort("X7"):  {"P5", "ADC15/GPIX7"},
		GpioPort("Y0"):  {"C21", "GPIOY0/SIOS3#"},
		GpioPort("Y1"):  {"F20", "GPIOY1/SIOS5#"},
		GpioPort("Y2"):  {"G20", "GPIOY2/SIOPWREQ#"},
		GpioPort("Y3"):  {"K20", "GPIOY3/SIOONCTRL#"},
		GpioPort("Z0"):  {"R22", "ROMA2/GPOZ0/VPOB0"},
		GpioPort("Z1"):  {"P18", "ROMA3/GPOZ1/VPOB1"},
		GpioPort("Z2"):  {"P19", "ROMA4/GPOZ2/VPOB2"},
		GpioPort("Z3"):  {"P20", "ROMA5/GPOZ3/VPOB3"},
		GpioPort("Z4"):  {"P21", "ROMA6/GPOZ4/VPOB4"},
		GpioPort("Z5"):  {"P22", "ROMA7/GPOZ5/VPOB5"},
		GpioPort("Z6"):  {"M19", "ROMA8/GPOZ6/VPOB6"},
		GpioPort("Z7"):  {"M20", "ROMA9/GPOZ7/VPOB7"},
		GpioPort("AA0"): {"M21", "ROMA10/GPOAA0/VPOG0"},
		GpioPort("AA1"): {"M22", "ROMA11/GPOAA1/VPOG1"},
		GpioPort("AA2"): {"L18", "ROMA12/GPOAA2/VPOG2"},
		GpioPort("AA3"): {"L19", "ROMA13/GPOAA3/VPOG3"},
		GpioPort("AA4"): {"L20", "ROMA14/GPOAA4/VPOG4"},
		GpioPort("AA5"): {"L21", "ROMA15/GPOAA5/VPOG5"},
		GpioPort("AA6"): {"T18", "ROMA16/GPOAA6/VPOG6"},
		GpioPort("AA7"): {"N18", "ROMA17/GPOAA7/VPOG7"},
		GpioPort("AB0"): {"N19", "ROMA18/GPOAB0/VPOR0"},
		GpioPort("AB1"): {"M18", "ROMA19/GPOAB1/VPOR1"},
		GpioPort("AB2"): {"N22", "ROMA20/GPOAB2/VPOR2"},
		GpioPort("AB3"): {"N20", "ROMA21/GPOAB3/VPOR3"},
	}

	gpioDirRegs = map[uint32]gpioReg{
		0x004: {[]gpioRegSet{{"A", 8}, {"B", 8}, {"C", 8}, {"D", 8}}},
		0x024: {[]gpioRegSet{{"E", 8}, {"F", 8}, {"G", 8}, {"H", 8}}},
		0x074: {[]gpioRegSet{{"I", 8}, {"J", 8}, {"K", 8}, {"L", 8}}},
		0x07C: {[]gpioRegSet{{"M", 8}, {"N", 8}, {"O", 8}, {"P", 8}}},
		0x084: {[]gpioRegSet{{"Q", 8}, {"R", 8}, {"S", 8}, {"T", 8}}},
		0x08C: {[]gpioRegSet{{"U", 8}, {"V", 8}}},
		0x1E4: {[]gpioRegSet{{"Y", 4}}},
	}

	gpioDataRegs = map[uint32]gpioReg{
		0x000: {[]gpioRegSet{{"A", 8}, {"B", 8}, {"C", 8}, {"D", 8}}},
		0x020: {[]gpioRegSet{{"E", 8}, {"F", 8}, {"G", 8}, {"H", 8}}},
		0x070: {[]gpioRegSet{{"I", 8}, {"J", 8}, {"K", 8}, {"L", 8}}},
		0x078: {[]gpioRegSet{{"M", 8}, {"N", 8}, {"O", 8}, {"P", 8}}},
		0x080: {[]gpioRegSet{{"Q", 8}, {"R", 8}, {"S", 8}, {"T", 8}}},
		0x088: {[]gpioRegSet{{"U", 8}, {"V", 8}, {"W", 8}, {"X", 8}}},
		0x1E0: {[]gpioRegSet{{"Y", 4}, {"", 4}, {"Z", 8}, {"AA", 8}, {"AB", 4}}},
	}

	// List of SCUs that could be remotely interesting for GPIO purposes
	scuGpioRegs = []uint32{
		0x08, 0x0c, 0x10, 0x14, 0x18, 0x1c, 0x20, 0x24, 0x28,
		0x2c, 0x30, 0x34, 0x38, 0x3c, 0x4c, 0x70, 0x74, 0x7c,
		0x80, 0x84, 0x88, 0x8c, 0x90, 0x94, 0x9c, 0xa0, 0xa4,
		0xa8, 0xc0, 0xc4, 0xd0,
	}

	LINE_STATE_INPUT         = 0
	LINE_STATE_OUTPUT        = 1
	LINE_STATE_BECAME_INPUT  = 2
	LINE_STATE_BECAME_OUTPUT = 3
	LINE_STATE_LOW           = 4
	LINE_STATE_HIGH          = 5
	// Since this is a sampled system let's not call it rising edge/falling edge
	LINE_STATE_BECAME_LOW  = 6
	LINE_STATE_BECAME_HIGH = 7
	// SCUs control many aspects of what pins do, so track them
	LINE_STATE_SCU         = 8
	LINE_STATE_SCU_CHANGED = 9
)

type State struct {
	Gpio map[uint32]uint32
	Scu  map[uint32]uint32
}

func (s *State) List() []LineState {
	return s.diff(nil)
}

func (s *State) diff(b *State) []LineState {
	dirs := make(map[uint32]bool)
	res := make([]LineState, 0)

	for a, r := range gpioDirRegs {
		bo := 0
		for _, set := range r.sets {
			for pin := 0; pin < set.pins; pin++ {
				bit := uint(bo + pin)
				port := setPinToPort(set.set, pin)
				output := (s.Gpio[a] & (1 << bit)) != 0
				dirs[port] = output
				if b == nil {
					if output {
						res = append(res, LineState{port, LINE_STATE_OUTPUT})
					} else {
						res = append(res, LineState{port, LINE_STATE_INPUT})
					}
				}
				if b != nil && output != ((b.Gpio[a]&(1<<bit)) != 0) {
					if output {
						res = append(res, LineState{port, LINE_STATE_BECAME_OUTPUT})
					} else {
						res = append(res, LineState{port, LINE_STATE_BECAME_INPUT})
					}
				}
			}
			bo += set.pins
		}
	}

	for a, r := range gpioDataRegs {
		bo := 0
		for _, set := range r.sets {
			for pin := 0; pin < set.pins && set.set != ""; pin++ {
				bit := uint(bo + pin)
				port := setPinToPort(set.set, pin)
				high := (s.Gpio[a] & (1 << bit)) != 0
				if b == nil {
					if high {
						res = append(res, LineState{port, LINE_STATE_HIGH})
					} else {
						res = append(res, LineState{port, LINE_STATE_LOW})
					}
				} else if high != ((b.Gpio[a] & (1 << bit)) != 0) {
					if high {
						res = append(res, LineState{port, LINE_STATE_BECAME_HIGH})
					} else {
						res = append(res, LineState{port, LINE_STATE_BECAME_LOW})
					}
				}
			}
			bo += set.pins
		}
	}

	for _, scu := range scuGpioRegs {
		if b == nil {
			res = append(res, LineState{scu, LINE_STATE_SCU})
		} else if s.Scu[scu] != b.Scu[scu] {
			res = append(res, LineState{scu, LINE_STATE_SCU_CHANGED})
		}
	}

	return res
}

func (s *State) PortValue(port uint32) bool {
	tset, tpin := portToSetPin(port)
	for a, r := range gpioDataRegs {
		bo := 0
		for _, set := range r.sets {
			if set.set != tset {
				bo += set.pins
				continue
			}
			bit := uint(bo + tpin)
			high := (s.Gpio[a] & (1 << bit)) != 0
			return high
		}
	}
	panic("Unknown port")
}

func (s *State) Equal(b *State) bool {
	for r, c := range b.Gpio {
		v, ok := s.Gpio[r]
		if !ok {
			return false
		}
		if v != c {
			return false
		}
	}
	for r, c := range b.Scu {
		v, ok := s.Scu[r]
		if !ok {
			return false
		}
		if v != c {
			return false
		}
	}
	return true
}

func (s *State) Diff(b *State) []LineState {
	return s.diff(b)
}

func (a *Ast) SnapshotGpio() *State {
	base := uintptr(0x1e780000)

	s := State{}
	s.Gpio = make(map[uint32]uint32)
	s.Scu = make(map[uint32]uint32)
	for r, _ := range gpioDirRegs {
		s.Gpio[r] = a.Mem().MustRead32(base + uintptr(r))
	}
	for r, _ := range gpioDataRegs {
		s.Gpio[r] = a.Mem().MustRead32(base + uintptr(r))
	}
	for _, r := range scuGpioRegs {
		s.Scu[r] = a.Mem().MustRead32(SCU_BASE + uintptr(r))
	}
	return &s
}

// Resolve a GPIO name such as "A8" to the Linux GPIO line index
func GpioPort(n string) uint32 {
	n = strings.ToUpper(n)
	idx := uint32(0)
	off := -1
	if strings.HasPrefix(n, "AA") {
		idx = 26 * 8
		off = 2
	} else if strings.HasPrefix(n, "AB") {
		idx = 27 * 8
		off = 2
	} else {
		idx = uint32(n[0]-'A') * 8
		off = 1
	}
	o, err := strconv.ParseUint(n[off:], 10, 32)
	if err != nil {
		log.Fatalf("Unknown GPIO name: %s", n)
	}
	idx += uint32(o)
	return idx
}

func portToSetPin(p uint32) (string, int) {
	if p >= 27*8 {
		return "AB", int(p - 27*8)
	} else if p >= 26*8 {
		return "AA", int(p - 26*8)
	} else {
		return fmt.Sprintf("%c", 'A'+p/8), int(p % 8)
	}
}

func GpioPortToName(p uint32) string {
	set, pin := portToSetPin(p)
	return fmt.Sprintf("%s%d", set, pin)
}

func GpioPortToFunction(p uint32) string {
	return gpios[p].name
}

func setPinToPort(set string, port int) uint32 {
	if set == "AB" {
		return uint32(27*8 + port)
	} else if set == "AA" {
		return uint32(26*8 + port)
	} else {
		return uint32(int(set[0]-'A')*8 + port)
	}
}
