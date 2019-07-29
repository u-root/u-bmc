// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package platform

import (
	"log"
	"time"

	"github.com/u-root/u-bmc/pkg/aspeed"
	"github.com/u-root/u-bmc/pkg/bmc"
	"github.com/u-root/u-bmc/platform/quanta-f06-leopard-ddr3/pkg/gpio"

	pb "github.com/u-root/u-bmc/proto"
)

type platform struct {
	a *aspeed.Ast
	g *bmc.GpioSystem
	gpio.Gpio
}

func (p *platform) InitializeGpio(g *bmc.GpioSystem) error {
	p.g = g
	g.Monitor(map[string]bmc.GpioCallback{
		"CPU0_FIVR_FAULT_N":   bmc.LogGpio,
		"CPU0_PROCHOT_N":      bmc.LogGpio,
		"CPU0_THERMTRIP_N":    bmc.LogGpio,
		"CPU1_FIVR_FAULT_N":   bmc.LogGpio,
		"CPU1_PROCHOT_N":      bmc.LogGpio,
		"CPU1_THERMTRIP_N":    bmc.LogGpio,
		"CPU_CATERR_N":        bmc.LogGpio,
		"MB_SLOT_ID":          bmc.LogGpio,
		"MEMAB_MEMHOT_N":      bmc.LogGpio,
		"MEMCD_MEMHOT_N":      bmc.LogGpio,
		"MEMEF_MEMHOT_N":      bmc.LogGpio,
		"MEMGH_MEMHOT_N":      bmc.LogGpio,
		"NMI_BTN_N":           bmc.LogGpio,
		"PCH_BMC_THERMTRIP_N": bmc.LogGpio,
		"PCH_PWR_OK":          bmc.LogGpio,
		"PWR_BTN_N":           p.PowerButtonHandler,
		"RST_BTN_N":           p.ResetButtonHandler,
		"SKU0":                bmc.LogGpio,
		"SKU1":                bmc.LogGpio,
		"SKU2":                bmc.LogGpio,
		"SKU3":                bmc.LogGpio,
		"SLP_S3_N":            bmc.LogGpio,
		"SPI_SEL":             bmc.LogGpio,
		"SYS_PWR_OK":          bmc.LogGpio,
		"SYS_THROTTLE":        bmc.LogGpio,
		"UART_SELECT0":        bmc.LogGpio,
		"UART_SELECT1":        bmc.LogGpio,
		"UNKN_BOOT0":          bmc.LogGpio,
		"UNKN_BOOT1":          bmc.LogGpio,
	})

	g.Hog(map[string]bool{
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

	go g.ManageButton("BMC_PWR_BTN_OUT_N", pb.Button_BUTTON_POWER, bmc.GPIO_INVERTED)
	go g.ManageButton("BMC_RST_BTN_OUT_N", pb.Button_BUTTON_RESET, bmc.GPIO_INVERTED)
	return nil
}

func (p *platform) PowerButtonHandler(_ string, c chan bool, _ bool) {
	pushc := chan bool(nil)
	for state := range c {
		// Power button is inverted
		pressed := !state
		if pressed {
			log.Printf("Physical power button pressed")
			pushc = make(chan bool)
			p.g.Button(pb.Button_BUTTON_POWER) <- pushc
			pushc <- true
		} else if pushc != nil {
			log.Printf("Physical power button released")
			pushc <- false
			close(pushc)
			pushc = nil
		}
	}
}

func (p *platform) ResetButtonHandler(_ string, c chan bool, _ bool) {
	for state := range c {
		// Reset button is inverted
		pressed := !state
		if pressed {
			log.Printf("Physical reset button triggered")
			pushc := make(chan bool)
			p.g.Button(pb.Button_BUTTON_RESET) <- pushc
			pushc <- true
			time.Sleep(time.Duration(100) * time.Millisecond)
			pushc <- false
			close(pushc)
		}
	}
}

func (p *platform) InitializeSystem() error {
	// Configure UART routing:
	// - Route UART2 to UART3
	// - Route UART3 to UART2
	// TODO(bluecmd): Platform dependent
	p.a.Mem().MustWrite32(0x1E789000+0x9c, 0x6<<22|0x4<<19)

	// Re-enable the clock of UART2 to enable the internal routing
	// which will make u-bmc end of the pipe be /dev/ttyS2
	// This can be done by defining the uart2 as active in the dts, but
	// if we do that then /dev/ttyS1 might be confusing as it will not work
	// properly.
	p.a.Mem().MustWrite32(aspeed.SCU_BASE+0x0, aspeed.SCU_PASSWORD)
	csr := p.a.Mem().MustRead32(aspeed.SCU_BASE + 0x0c)
	p.a.Mem().MustWrite32(aspeed.SCU_BASE+0x0c, csr & ^uint32(1<<16))
	// Enable UART1 and UART2 pins
	mfr := p.a.Mem().MustRead32(aspeed.SCU_BASE + 0x84)
	p.a.Mem().MustWrite32(aspeed.SCU_BASE+0x84, mfr|0xffff0000)
	// Disable all pass-through GPIO ports. This enables u-bmc to control
	// the power buttons, which are routed as pass-through before boot has
	// completed.
	hws := p.a.Mem().MustRead32(aspeed.SCU_BASE + 0x70)
	p.a.Mem().MustWrite32(aspeed.SCU_BASE+0x70, hws & ^uint32(3<<21))
	p.a.Mem().MustWrite32(aspeed.SCU_BASE+0x8c, 0)
	p.a.Mem().MustWrite32(aspeed.SCU_BASE+0x0, 0x0)

	log.Printf("Setting up Network Controller Sideband Interface (NC-SI) for eth0")
	go bmc.StartNcsi("eth0")
	return nil
}

func (p *platform) PwmMap() map[int]string {
	return map[int]string{
		0: "/sys/class/hwmon/hwmon0/pwm1",
		1: "/sys/class/hwmon/hwmon0/pwm2",
	}
}

func (p *platform) FanMap() map[int]string {
	return map[int]string{
		0: "/sys/class/hwmon/hwmon0/fan1_input",
		1: "/sys/class/hwmon/hwmon0/fan3_input",
	}
}

func (p *platform) ThermometerMap() map[int]string {
	// TODO(bluecmd): These are unverified, most likely there are more
	return map[int]string{
		0: "/sys/class/hwmon/hwmon1/temp1_input",
		1: "/sys/class/hwmon/hwmon2/temp1_input",
	}
}

func (p *platform) HostUart() (string, int) {
	return "/dev/ttyS2", 57600
}

func (p *platform) Close() {
	p.a.Close()
}

func Platform() *platform {
	a := aspeed.Open()
	p := platform{a, nil, gpio.Gpio{}}
	return &p
}
