// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package platform

import (
	"testing"
	"time"

	"github.com/u-root/u-bmc/pkg/bmc"
)

var (
	powerButtonN uint32
	powerOutN    uint32
	resetButtonN uint32
	resetOutN    uint32
	sysPwrOK     uint32
	powerLed     uint32
)

func TestMain(t *testing.T) {
	var ok bool
	p := platform{}
	powerButtonN, ok = p.GpioNameToPort("PWR_BTN_N")
	if !ok {
		t.Fatalf("Button PWR_BTN_N not defined")
	}
	powerOutN, ok = p.GpioNameToPort("BMC_PWR_BTN_OUT_N")
	if !ok {
		t.Fatalf("Button BMC_PWR_BTN_OUT_N not defined")
	}
	resetButtonN, ok = p.GpioNameToPort("RST_BTN_N")
	if !ok {
		t.Fatalf("Button RST_BTN_N not defined")
	}
	resetOutN, ok = p.GpioNameToPort("BMC_RST_BTN_OUT_N")
	if !ok {
		t.Fatalf("Button BMC_RST_BTN_OUT_N not defined")
	}
	sysPwrOK, ok = p.GpioNameToPort("SYS_PWR_OK")
	if !ok {
		t.Fatalf("Port SYS_PWR_OK not defined")
	}
	powerLed, ok = p.GpioNameToPort("PWR_LED")
	if !ok {
		t.Fatalf("Port PWR_LED not defined")
	}
}

func TestPowerButton(t *testing.T) {
	p := platform{}
	f := bmc.FakeGpioImpl(&p, map[uint32]bool{
		// Button is inverted, default is high
		powerButtonN: true,
		powerOutN:    true,
		resetButtonN: true,
		resetOutN:    true,
	})

	g := bmc.NewGpioSystem(&p, f)
	err := p.InitializeGpio(g)
	if err != nil {
		t.Fatalf("platform.InitializeGpio failed with %v", err)
	}

	// Power button out mirrors the power button press
	if !f.Get(powerOutN) {
		t.Fatalf("Power control line low when power button is in resting state")
	}

	f.Set(powerButtonN, false)
	if f.Get(powerOutN) {
		t.Fatalf("Power control line remained high when power button is being pushed")
	}

	f.Set(powerButtonN, true)
	if !f.Get(powerOutN) {
		t.Fatalf("Power control line remained low when power button was released")
	}
}

func TestResetButton(t *testing.T) {
	p := platform{}
	f := bmc.FakeGpioImpl(&p, map[uint32]bool{
		// Button is inverted, default is high
		powerButtonN: true,
		powerOutN:    true,
		resetButtonN: true,
		resetOutN:    true,
	})

	g := bmc.NewGpioSystem(&p, f)
	err := p.InitializeGpio(g)
	if err != nil {
		t.Fatalf("platform.InitializeGpio failed with %v", err)
	}

	// Reset press causes a 100 ms pulse
	if !f.Get(resetOutN) {
		t.Fatalf("Reset control line low when reset button is in resting state")
	}

	f.Set(resetButtonN, false)
	if f.Get(resetOutN) {
		t.Fatalf("Reset control line remained high when reset button is being pushed")
	}

	f.Set(resetButtonN, true)
	if f.Get(resetOutN) {
		t.Fatalf("Reset control line did not remain low when reset button was released")
	}

	// TODO(bluecmd): This should be using a fake clock to avoid races and long tests.
	time.Sleep(time.Duration(110) * time.Millisecond)
	if !f.Get(resetOutN) {
		t.Fatalf("Reset control line did not release after 100 ms")
	}
}

func TestPowerLED(t *testing.T) {
	p := platform{powerLed: make(chan bool)}
	f := bmc.FakeGpioImpl(&p, map[uint32]bool{
		sysPwrOK: false,
		powerLed: false,
	})

	g := bmc.NewGpioSystem(&p, f)
	err := p.InitializeGpio(g)
	if err != nil {
		t.Fatalf("platform.InitializeGpio failed with %v", err)
	}

	if f.Get(powerLed) {
		t.Fatalf("Power LED active when system is off")
	}

	f.Set(sysPwrOK, true)
	if !f.Get(powerLed) {
		t.Fatalf("Power LED remained inactive when system was turned on")
	}

	f.Set(sysPwrOK, false)
	if f.Get(powerLed) {
		t.Fatalf("Power LED remained active when system was turned off again")
	}
}
