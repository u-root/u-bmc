// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"

	"github.com/u-root/u-bmc/pkg/ast2400"
	"github.com/u-root/u-bmc/pkg/bmc"
	"github.com/u-root/u-bmc/platform/quanta-f06-leopard-ddr3/gpio"

	pb "github.com/u-root/u-bmc/proto"
)

type platform struct {
	a *ast2400.Ast
	gpio.Gpio
}

func (p *platform) InitializeGpio(g *bmc.GpioSystem) error {
	go g.Monitor([]string{
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

	go g.Hog(map[string]bool{
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

	go g.ManageButton("BMC_PWR_BTN_OUT_N", pb.Button_BUTTON_POWER)
	go g.ManageButton("BMC_RST_BTN_OUT_N", pb.Button_BUTTON_RESET)
	return nil
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
	p.a.Mem().MustWrite32(ast2400.SCU_BASE+0x0, ast2400.SCU_PASSWORD)
	csr := p.a.Mem().MustRead32(ast2400.SCU_BASE + 0x0c)
	p.a.Mem().MustWrite32(ast2400.SCU_BASE+0x0c, csr & ^uint32(1<<16))
	// Enable UART1 and UART2 pins
	mfr := p.a.Mem().MustRead32(ast2400.SCU_BASE + 0x84)
	p.a.Mem().MustWrite32(ast2400.SCU_BASE+0x84, mfr|0xffff0000)
	p.a.Mem().MustWrite32(ast2400.SCU_BASE+0x0, 0x0)

	log.Printf("Setting up Network Controller Sideband Interface (NC-SI) for eth0")
	go bmc.StartNcsi("eth0")
	return nil
}

func (p *platform) PwmMap() map[int]string {
	return map[int]string{
		0: "hwmon0/pwm1",
		1: "hwmon0/pwm2",
	}
}

func (p *platform) FanMap() map[int]string {
	return map[int]string{
		0: "hwmon0/fan1_input",
		1: "hwmon0/fan2_input",
	}
}

func (p *platform) HostUart() (string, int) {
	return "/dev/ttyS2", 57600
}

func (p *platform) InitializeFans(fan *bmc.FanSystem) error {
	for i := 0; i < fan.FanCount(); i++ {
		log.Printf("Configuring fan %d for 20%%", i)
		go fan.SetFanPercentage(i, 20)
	}
	return nil
}

func main() {
	a := ast2400.Open()
	defer a.Close()
	p := platform{a, gpio.Gpio{}}
	bmc.Startup(&p)
}
