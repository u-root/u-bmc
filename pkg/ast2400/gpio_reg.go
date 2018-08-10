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

var (
	gpios = map[string]gpio{
		"A0":  {"D6", "GPIOA0/MAC1LINK"},
		"A1":  {"B5", "GPIOA1/MAC2LINK"},
		"A2":  {"A4", "GPIOA2/TIMER3"},
		"A3":  {"E6", "GPIOA3/TIMER4"},
		"A4":  {"C5", "GPIOA4/TIMER5/SCL9"},
		"A5":  {"B4", "GPIOA5/TIMER6/SDA9"},
		"A6":  {"A3", "GPIOA6/TIMER7/MDC2"},
		"A7":  {"D5", "GPIOA7/TIMER8/MDIO2"},
		"B0":  {"J21", "GPIOB0/SALT1"},
		"B1":  {"J20", "GPIOB1/SALT2"},
		"B2":  {"H18", "GPIOB2/SALT3"},
		"B3":  {"F18", "GPIOB3/SALT4"},
		"B4":  {"E19", "GPIOB4/LPCRST#"},
		"B5":  {"H19", "GPIOB5/LPCPD#/LPCSMI#"},
		"B6":  {"H20", "GPIOB6/LPCPME#"},
		"B7":  {"E18", "GPIOB7/EXTRST#/SPICS1#"},
		"C0":  {"C4", "GPIOC0/SD1CLK/SCL10"},
		"C1":  {"B3", "GPIOC1/SD1CMD/SDA10"},
		"C2":  {"A2", "GPIOC2/SD1DAT0/SCL11"},
		"C3":  {"E5", "GPIOC3/SD1DAT1/SDA11"},
		"C4":  {"D4", "GPIOC4/SD1DAT2/SCL12"},
		"C5":  {"C3", "GPIOC5/SD1DAT3/SDA12"},
		"C6":  {"B2", "GPIOC6/SD1CD#/SCL13"},
		"C7":  {"A1", "GPIOC7/SD1WP#/SDA13"},
		"D0":  {"A18", "GPIOD0/SD2CLK"},
		"D1":  {"D16", "GPIOD1/SD2CMD"},
		"D2":  {"B17", "GPIOD2/SD2DAT0"},
		"D3":  {"A17", "GPIOD3/SD2DAT1"},
		"D4":  {"C16", "GPIOD4/SD2DAT2"},
		"D5":  {"B16", "GPIOD5/SD2DAT3"},
		"D6":  {"A16", "GPIOD6/SD2CD#"},
		"D7":  {"E15", "GPIOD7/SD2WP#"},
		"E0":  {"D15", "GPIOE0/NCTS3"},
		"E1":  {"C15", "GPIOE1/NDCD3"},
		"E2":  {"B15", "GPIOE2/NDSR3"},
		"E3":  {"A15", "GPIOE3/NRI3"},
		"E4":  {"E14", "GPIOE4/NDTR3"},
		"E5":  {"D14", "GPIOE5/NRTS3"},
		"E6":  {"C14", "GPIOE6/TXD3"},
		"E7":  {"B14", "GPIOE7/RXD3"},
		"F0":  {"D18", "GPIOF0/NCTS4"},
		"F1":  {"B19", "GPIOF1/NDCD4/SIOPBI#"},
		"F2":  {"A20", "GPIOF2/NDSR4/SIOPWRGD"},
		"F3":  {"D17", "GPIOF3/NRI4/SIOPBO#"},
		"F4":  {"B18", "GPIOF4/NDTR4"},
		"F5":  {"A19", "GPIOF5/NRTS4/SIOSCI#"},
		"F6":  {"E16", "GPIOF6/TXD4"},
		"F7":  {"C17", "GPIOF7/RXD4"},
		"G0":  {"A14", "GPIOG0/SGPSCK"},
		"G1":  {"E13", "GPIOG1/SGPSLD"},
		"G2":  {"D13", "GPIOG2/SGPSI0"},
		"G3":  {"C13", "GPIOG3/SGPSI1"},
		"G4":  {"B13", "GPIOG4/WDTRST1/OSCCLK"},
		"G5":  {"Y21", "GPIOG5/WDTRST2/USBCKI"},
		"G6":  {"AA22", "GPIOG6/FLBUSY#"},
		"G7":  {"U18", "GPIOG7/FLWP#"},
		"H0":  {"A8", "GPIOH0/ROMD8/NCTS6"},
		"H1":  {"C7", "GPIOH1/ROMD9/NDCD6"},
		"H2":  {"B7", "GPIOH2/ROMD10/NDSR6"},
		"H3":  {"A7", "GPIOH3/ROMD11/NRI6"},
		"H4":  {"D7", "GPIOH4/ROMD12/NDTR6"},
		"H5":  {"B6", "GPIOH5/ROMD13/NRTS6"},
		"H6":  {"A6", "GPIOH6/ROMD14/TXD6"},
		"H7":  {"E7", "GPIOH7/ROMD15/RXD6"},
		"I0":  {"C22", "GPIOI0/SYSCS#"},
		"I1":  {"G18", "GPIOI1/SYSCK"},
		"I2":  {"D19", "GPIOI2/SYSDO"},
		"I3":  {"C20", "GPIOI3/SYSDI"},
		"I4":  {"B22", "GPIOI4/SPICS#/VBCS#"},
		"I5":  {"G19", "GPIOI5/SPICK/VBCK"},
		"I6":  {"C18", "GPIOI6/SPIDO/VBDO"},
		"I7":  {"E20", "GPIOI7/SPIDI/VBDI"},
		"J0":  {"J5", "GPIOJ0/SGPMCK"},
		"J1":  {"J4", "GPIOJ1/SGPMLD"},
		"J2":  {"K5", "GPIOJ2/SGPMO"},
		"J3":  {"J3", "GPIOJ3/SGPMI"},
		"J4":  {"T4", "VGAHS/GPIOJ4"},
		"J5":  {"U2", "VGAVS/GPIOJ5"},
		"J6":  {"T2", "DDCCLK/GPIOJ6"},
		"J7":  {"T1", "DDCDAT/GPIOJ7"},
		"K0":  {"E3", "GPIOK0/SCL5"},
		"K1":  {"D2", "GPIOK1/SDA5"},
		"K2":  {"C1", "GPIOK2/SCL6"},
		"K3":  {"F4", "GPIOK3/SDA6"},
		"K4":  {"E2", "GPIOK4/SCL7"},
		"K5":  {"D1", "GPIOK5/SDA7"},
		"K6":  {"G5", "GPIOK6/SCL8"},
		"K7":  {"F3", "GPIOK7/SDA8"},
		"L0":  {"U1", "GPIOL0/NCTS1"},
		"L1":  {"T5", "GPIOL1/NDCD1/VPIDE"},
		"L2":  {"U3", "GPIOL2/NDSR1/VPIODD"},
		"L3":  {"V1", "GPIOL3/NRI1/VPIHS"},
		"L4":  {"U4", "GPIOL4/NDTR1/VPIVS"},
		"L5":  {"V2", "GPIOL5/NRTS1/VPICLK"},
		"L6":  {"W1", "GPIOL6/TXD1/VPIB0"},
		"L7":  {"U5", "GPIOL7/RXD1/VPIB1"},
		"M0":  {"V3", "GPIOM0/NCTS2/VPIB2"},
		"M1":  {"W2", "GPIOM1/NDCD2/VPIB3"},
		"M2":  {"Y1", "GPIOM2/NDSR2/VPIB4"},
		"M3":  {"V4", "GPIOM3/NRI2/VPIB5"},
		"M4":  {"W3", "GPIOM4/NDTR2/VPIB6"},
		"M5":  {"Y2", "GPIOM5/NRTS2/VPIB7"},
		"M6":  {"AA1", "GPIOM6/TXD2/VPIB8"},
		"M7":  {"V5", "GPIOM7/RXD2/VPIB9"},
		"N0":  {"W4", "GPION0/PWM0/VPIG0"},
		"N1":  {"Y3", "GPION1/PWM1/VPIG1"},
		"N2":  {"AA2", "GPION2/PWM2/VPIG2"},
		"N3":  {"AB1", "GPION3/PWM3/VPIG3"},
		"N4":  {"W5", "GPION4/PWM4/VPIG4"},
		"N5":  {"Y4", "GPION5/PWM5/VPIG5"},
		"N6":  {"AA3", "GPION6/PWM6/VPIG6"},
		"N7":  {"AB2", "GPION7/PWM7/VPIG7"},
		"O0":  {"V6", "GPIOO0/TACH0/VPIG8"},
		"O1":  {"Y5", "GPIOO1/TACH1/VPIG9"},
		"O2":  {"AA4", "GPIOO2/TACH2/VPIR0"},
		"O3":  {"AB3", "GPIOO3/TACH3/VPIR1"},
		"O4":  {"W6", "GPIOO4/TACH4/VPIR2"},
		"O5":  {"AA5", "GPIOO5/TACH5/VPIR3"},
		"O6":  {"AB4", "GPIOO6/TACH6/VPIR4"},
		"O7":  {"V7", "GPIOO7/TACH7/VPIR5"},
		"P0":  {"Y6", "GPIOP0/TACH8/VPIR6"},
		"P1":  {"AB5", "GPIOP1/TACH9/VPIR7"},
		"P2":  {"W7", "GPIOP2/TACH10/VPIR8"},
		"P3":  {"AA6", "GPIOP3/TACH11/VPIR9"},
		"P4":  {"AB6", "GPIOP4/TACH12"},
		"P5":  {"Y7", "GPIOP5/TACH13"},
		"P6":  {"AA7", "GPIOP6/TACH14/BMCINT"},
		"P7":  {"AB7", "GPIOP7/TACH15/FLACK"},
		"Q0":  {"D3", "GPIOQ0/SCL3"},
		"Q1":  {"C2", "GPIOQ1/SDA3"},
		"Q2":  {"B1", "GPIOQ2/SCL4"},
		"Q3":  {"F5", "GPIOQ3/SDA4"},
		"Q4":  {"H4", "GPIOQ4/SCL14"},
		"Q5":  {"H3", "GPIOQ5/SDA14"},
		"Q6":  {"H2", "GPIOQ6"},
		"Q7":  {"H1", "GPIOQ7"},
		"R0":  {"V20", "GPIOR0/ROMCS1#"},
		"R1":  {"W21", "GPIOR1/ROMCS2#"},
		"R2":  {"Y22", "GPIOR2/ROMCS3#"},
		"R3":  {"U19", "GPIOR3/ROMCS4#"},
		"R4":  {"V21", "GPIOR4/ROMA24/VPOR6"},
		"R5":  {"W22", "GPIOR5/ROMA25/VPOR7"},
		"R6":  {"C6", "GPIOR6/MDC1"},
		"R7":  {"A5", "GPIOR7/MDIO1"},
		"S0":  {"U21", "ROMD4/GPIOS0/VPODE"},
		"S1":  {"T19", "ROMD5/GPIOS1/VPOHS"},
		"S2":  {"V22", "ROMD6/GPIOS2/VPOVS"},
		"S3":  {"U20", "ROMD7/GPIOS3/VPOCLK"},
		"S4":  {"R18", "ROMOE#/GPIOS4"},
		"S5":  {"N21", "ROMWE#/GPIOS5"},
		"S6":  {"L22", "ROMA22/GPIOS6/VPOR4"},
		"S7":  {"K18", "ROMA23/GPIOS7/VPOR5"},
		"T0":  {"A12", "RGMII1TXCK/RMII1TXEN/GPIOT0"},
		"T1":  {"B12", "RGMII1TXCTL/GPIOT1"},
		"T2":  {"C12", "RGMII1TXD0/RMII1TXD0/GPIOT2"},
		"T3":  {"D12", "RGMII1TXD1/RMII1TXD1/GPIOT3"},
		"T4":  {"E12", "RGMII1TXD2/GPIOT4"},
		"T5":  {"A13", "RGMII1TXD3/GPIOT5"},
		"T6":  {"D9", "RGMII2TXCK/RMII2TXEN/GPIOT6"},
		"T7":  {"E9", "RGMII2TXCTL/GPIOT7"},
		"U0":  {"A10", "RGMII2TXD0/RMII2TXD0/GPIOU0"},
		"U1":  {"B10", "RGMII2TXD1/RMII2TXD1/GPIOU1"},
		"U2":  {"C10", "RGMII2TXD2/GPIOU2"},
		"U3":  {"D10", "RGMII2TXD3/GPIOU3"},
		"U4":  {"E11", "RGMII1RXCK/RMII1RCLK/GPIOU4"},
		"U5":  {"D11", "RGMII1RXCTL/GPIOU5"},
		"U6":  {"C11", "RGMII1RXD0/RMII1RXD0/GPIOU6"},
		"U7":  {"B11", "RGMII1RXD1/RMII1RXD1/GPIOU7"},
		"V0":  {"A11", "RGMII1RXD2/RMII1CRSDV/GPIOV0"},
		"V1":  {"E10", "RGMII1RXD3/RMII1RXER/GPIOV1"},
		"V2":  {"C9", "RGMII2RXCK/RMII2RCLK/GPIOV2"},
		"V3":  {"B9", "RGMII2RXCTL/GPIOV3"},
		"V4":  {"A9", "RGMII2RXD0/RMII2RXD0/GPIOV4"},
		"V5":  {"E8", "RGMII2RXD1/RMII2RXD1/GPIOV5"},
		"V6":  {"D8", "RGMII2RXD2/RMII2CRSDV/GPIOV6"},
		"V7":  {"C8", "RGMII2RXD3/RMII2RXER/GPIOV7"},
		"W0":  {"L5", "ADC0/GPIW0"},
		"W1":  {"L4", "ADC1/GPIW1"},
		"W2":  {"L3", "ADC2/GPIW2"},
		"W3":  {"L2", "ADC3/GPIW3"},
		"W4":  {"L1", "ADC4/GPIW4"},
		"W5":  {"M5", "ADC5/GPIW5"},
		"W6":  {"M4", "ADC6/GPIW6"},
		"W7":  {"M3", "ADC7/GPIW7"},
		"X0":  {"M2", "ADC8/GPIX0"},
		"X1":  {"M1", "ADC9/GPIX1"},
		"X2":  {"N5", "ADC10/GPIX2"},
		"X3":  {"N4", "ADC11/GPIX3"},
		"X4":  {"N3", "ADC12/GPIX4"},
		"X5":  {"N2", "ADC13/GPIX5"},
		"X6":  {"N1", "ADC14/GPIX6"},
		"X7":  {"P5", "ADC15/GPIX7"},
		"Y0":  {"C21", "GPIOY0/SIOS3#"},
		"Y1":  {"F20", "GPIOY1/SIOS5#"},
		"Y2":  {"G20", "GPIOY2/SIOPWREQ#"},
		"Y3":  {"K20", "GPIOY3/SIOONCTRL#"},
		"Z0":  {"R22", "ROMA2/GPOZ0/VPOB0"},
		"Z1":  {"P18", "ROMA3/GPOZ1/VPOB1"},
		"Z2":  {"P19", "ROMA4/GPOZ2/VPOB2"},
		"Z3":  {"P20", "ROMA5/GPOZ3/VPOB3"},
		"Z4":  {"P21", "ROMA6/GPOZ4/VPOB4"},
		"Z5":  {"P22", "ROMA7/GPOZ5/VPOB5"},
		"Z6":  {"M19", "ROMA8/GPOZ6/VPOB6"},
		"Z7":  {"M20", "ROMA9/GPOZ7/VPOB7"},
		"AA0": {"M21", "ROMA10/GPOAA0/VPOG0"},
		"AA1": {"M22", "ROMA11/GPOAA1/VPOG1"},
		"AA2": {"L18", "ROMA12/GPOAA2/VPOG2"},
		"AA3": {"L19", "ROMA13/GPOAA3/VPOG3"},
		"AA4": {"L20", "ROMA14/GPOAA4/VPOG4"},
		"AA5": {"L21", "ROMA15/GPOAA5/VPOG5"},
		"AA6": {"T18", "ROMA16/GPOAA6/VPOG6"},
		"AA7": {"N18", "ROMA17/GPOAA7/VPOG7"},
		"AB0": {"N19", "ROMA18/GPOAB0/VPOR0"},
		"AB1": {"M18", "ROMA19/GPOAB1/VPOR1"},
		"AB2": {"N22", "ROMA20/GPOAB2/VPOR2"},
		"AB3": {"N20", "ROMA21/GPOAB3/VPOR3"},
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
)

type state struct {
	r map[uint32]uint32
}

func (s *state) Print() {
	s.printDiff(nil)
}

func (s *state) printDiff(b *state) {
	dirs := make(map[string]bool)

	for a, r := range gpioDirRegs {
		bo := 0
		for _, set := range r.sets {
			for pin := 0; pin < set.pins; pin++ {
				bit := uint(bo + pin)
				name := fmt.Sprintf("%s%d", set.set, pin)
				g, ok := gpios[name]
				if !ok {
					panic("Unknown GPIO calculated: " + name)
				}
				output := (s.r[a] & (1 << bit)) != 0
				dirs[name] = output
				if output && b == nil {
					fmt.Printf("%s is output\n", g.name)
					continue
				}
				if b != nil && output != ((b.r[a]&(1<<bit)) != 0) {
					if output {
						log.Printf("%s became output\n", g.name)
					} else {
						log.Printf("%s became input\n", g.name)
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
				name := fmt.Sprintf("%s%d", set.set, pin)
				g, ok := gpios[name]
				if !ok {
					panic("Unknown GPIO calculated: " + name)
				}
				high := (s.r[a] & (1 << bit)) != 0
				if b == nil {
					if high {
						log.Printf("%30s X (output: %v)\n", g.name, dirs[name])
					} else {
						log.Printf("%30s 0 (output: %v)\n", g.name, dirs[name])
					}
				} else if high != ((b.r[a] & (1 << bit)) != 0) {
					if high {
						log.Printf("%30s 0->X (output: %v)\n", g.name, dirs[name])
					} else {
						log.Printf("%30s X->0 (output: %v)\n", g.name, dirs[name])
					}
				}
			}
			bo += set.pins
		}
	}

}

func (s *state) Equals(b *state) bool {
	if len(b.r) != len(s.r) {
		return false
	}
	for r, c := range b.r {
		v, ok := s.r[r]
		if !ok {
			return false
		}
		if v != c {
			return false
		}
	}
	return true
}

func (s *state) Diff(b *state) {
	s.printDiff(b)
}

func (a *Ast) SnapshotGpio() *state {
	base := uintptr(0x1e780000)

	s := state{}
	s.r = make(map[uint32]uint32)
	for r, _ := range gpioDirRegs {
		s.r[r] = a.Mem().MustRead32(base + uintptr(r))
	}
	for r, _ := range gpioDataRegs {
		s.r[r] = a.Mem().MustRead32(base + uintptr(r))
	}
	return &s
}

// Resolve a GPIO name such as "A8" to the Linux GPIO line index
func GpioPort(n string) uint32 {
	n = strings.ToUpper(n)
	idx := uint32(0)
	if strings.HasPrefix(n, "AA") {
		idx = 26 * 8
	} else if strings.HasPrefix(n, "AB") {
		idx = 27 * 8
	} else {
		idx = uint32(n[0]-'A') * 8
	}
	o, err := strconv.ParseUint(n[1:], 10, 32)
	if err != nil {
		log.Fatalf("Unknown GPIO name: %s", n)
	}
	idx += uint32(o)
	return idx
}
