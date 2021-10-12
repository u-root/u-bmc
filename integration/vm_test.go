// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package integration

import (
	"testing"
	"time"

	"github.com/anatol/vmtest"
)

var (
	machine = []string{"-machine", "virt"}
	cpu32   = []string{"-cpu", "cortex-a15"}
	cpu64   = []string{"-cpu", "cortex-a72"}
	mem     = []string{"-m", "512"}
	blkdev  = []string{"-drive", "file=../build/img/rootfs.img,format=raw,if=virtio"}
	nic     = []string{"-nic", "user,hostfwd=udp::6053-:53,hostfwd=tcp::6443-:443,hostfwd=tcp::9370-:9370,model=virtio"}
	rng     = []string{"-device", "virtio-rng"}
)

type QOMInteger struct {
	Path     string `json:"path"`
	Property string `json:"property"`
	Value    int    `json:"value"`
}

type TestVM struct {
	cleanup func()
	*vmtest.Qemu
}

func cmdline(args ...[]string) (cmdline []string) {
	for _, arg := range args {
		cmdline = append(cmdline, arg...)
	}
	return
}

func NewTestVM(t *testing.T, bit int, timeout time.Duration, cleanup func()) (*TestVM, error) {
	var cpu []string
	var arch string
	switch bit {
	case 32:
		cpu = cpu32
		arch = "arm"
	case 64:
		cpu = cpu64
		arch = "aarch64"
	default:
		t.Fatalf("invalid arch: %dbit", bit)
	}
	opts := vmtest.QemuOptions{
		OperatingSystem: vmtest.OS_OTHER,
		Architecture:    vmtest.QemuArchitecture(arch),
		Kernel:          "../build/linux/zImage.boot",
		Params: cmdline(
			machine,
			cpu,
			mem,
			blkdev,
			nic,
			rng,
		),
		Verbose: testing.Verbose(),
		Timeout: timeout,
	}
	vm, err := vmtest.NewQemu(&opts)
	if err != nil {
		return nil, err
	}

	return &TestVM{cleanup, vm}, nil
}

func (v *TestVM) Close() {
	v.Qemu.Kill()
	v.cleanup()
}
