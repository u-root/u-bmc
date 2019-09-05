// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build amd64

package integration

import (
	"testing"

	"github.com/u-root/u-root/pkg/uroot"
)

func reportTemperature(t *testing.T, vm *TestVM, celcius int) {
	// TODO(bluecmd): Right now I don't know which sensor is what, so naming
	// them doesn't make much sense. We will very likely want to test interactions
	// on all sensors in the future though, so this will need to change.
	err := vm.SetQOMInteger("/machine/soc/i2c/aspeed.i2c.6/child[0]", "temperature0", celcius*1000)
	if err != nil {
		t.Fatalf("Unable to set temperature: %v", err)
	}
}

func waitFor(t *testing.T, vm *TestVM, marker string) {
	if err := vm.Expect(marker); err != nil {
		t.Fatalf(`expected "%s", got error: %v`, marker, err)
	}
}

// TestTempSensor checks the temperature sensor is hooked up properly
func TestTempSensor(t *testing.T) {
	bmc, bmccleanup := BMCTest(t, &Options{
		Name: "TestTempSensor-BMC",
		BuildOpts: uroot.Opts{
			Commands: uroot.BusyBoxCmds(
				"github.com/u-root/u-root/cmds/core/init",
				"github.com/u-root/u-bmc/integration/testcmd/tempsensor/uinit",
			),
		},
	})
	defer bmccleanup()

	waitFor(t, bmc, "TEST_SET_NORMAL_TEMP")
	reportTemperature(t, bmc, 23)

	waitFor(t, bmc, "TEST_SET_HIGH_TEMP")
	reportTemperature(t, bmc, 100)

	waitFor(t, bmc, "TEST_SET_LOW_TEMP")
	reportTemperature(t, bmc, 0)

	if err := bmc.Expect("TEST_OK"); err != nil {
		t.Fatal(`expected "TEST_OK", got error: `, err)
	}
}
