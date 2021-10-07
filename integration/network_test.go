// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64

package integration

import (
	"testing"

	"github.com/u-root/u-root/pkg/qemu"
	"github.com/u-root/u-root/pkg/uroot"
)

// Tries to access /metrics from a remote host
func TestMetrics(t *testing.T) {
	network := qemu.NewNetwork()
	_, bmccleanup := BMCTest(t, &Options{
		Name: "TestMetrics-BMC",
		BuildOpts: uroot.Opts{
			Commands: uroot.BusyBoxCmds(
				"github.com/u-root/u-root/cmds/core/init",
				"github.com/u-root/u-bmc/integration/testcmd/noop/uinit",
			),
		},
		QEMUOpts: qemu.Options{
			Devices: []qemu.Device{network.NewVM()},
		},
	})
	defer bmccleanup()

	host, hostcleanup := NativeTest(t, &Options{
		Name: "TestMetrics-Host",
		BuildOpts: uroot.Opts{
			Commands: uroot.BusyBoxCmds(
				"github.com/u-root/u-root/cmds/core/init",
				"github.com/u-root/u-root/cmds/core/wget",
				// TODO(bluecmd): This could be a "Uinit" script probably when the u-root
				// integration suite is a package
				"github.com/u-root/u-bmc/integration/testcmd/metrics-native/uinit",
			),
		},
		QEMUOpts: qemu.Options{
			Devices: []qemu.Device{network.NewVM()},
		},
	})
	defer hostcleanup()

	if err := host.Expect("TEST_OK"); err != nil {
		t.Fatal(`expected "TEST_OK" on host, got error: `, err)
	}
}
