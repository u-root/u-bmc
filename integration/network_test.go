// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build amd64

package integration

import (
	"testing"

	"github.com/u-root/u-root/pkg/qemu"
)

// Tries to access /metrics from a remote host
func TestMetrics(t *testing.T) {
	network := qemu.NewNetwork()
	_, bmccleanup := BMCTest(t, &Options{
		Name: "TestMetrics-BMC",
		Cmds: []string{
			"github.com/u-root/u-root/cmds/init",
			"github.com/u-root/u-bmc/integration/testcmd/noop/uinit",
		},
		Network: network,
	})
	defer bmccleanup()

	host, hostcleanup := NativeTest(t, &Options{
		Name: "TestMetrics-Host",
		Cmds: []string{
			"github.com/u-root/u-root/cmds/init",
			"github.com/u-root/u-root/cmds/wget",
			// TODO(bluecmd): This could be a "Uinit" script probably when the u-root
			// integration suite is a package
			"github.com/u-root/u-bmc/integration/testcmd/metrics-native/uinit",
		},
		Network: network,
	})
	defer hostcleanup()

	if err := host.Expect("TEST_OK"); err != nil {
		t.Fatal(`expected "TEST_OK" on host, got error: `, err)
	}
}
