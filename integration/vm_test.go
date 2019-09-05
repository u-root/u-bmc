// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package integration

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/digitalocean/go-qemu/qmp"
	"github.com/u-root/u-root/pkg/qemu"
)

type FlashDevice struct {
	Image string
}

func (d FlashDevice) Cmdline() []string {
	return []string{"-drive", "file=" + d.Image + ",format=raw,if=mtd"}
}

func (d FlashDevice) KArgs() []string {
	return nil
}

type MemoryDevice struct {
	MB int
}

func (d MemoryDevice) Cmdline() []string {
	return []string{"-m", fmt.Sprintf("%d", d.MB)}
}

func (d MemoryDevice) KArgs() []string {
	return nil
}

type MachineDevice struct {
	Board string
}

func (d MachineDevice) Cmdline() []string {
	return []string{"-M", d.Board}
}

func (d MachineDevice) KArgs() []string {
	return nil
}

type QemuMonitorDevice struct {
	Socket string
}

func (d QemuMonitorDevice) Cmdline() []string {
	return []string{"-qmp", "unix:" + d.Socket + ",server,nowait"}
}

func (d QemuMonitorDevice) KArgs() []string {
	return nil
}

type VirtioRngDevice struct {
}

func (d VirtioRngDevice) Cmdline() []string {
	return []string{"-device", "virtio-rng-pci"}
}

func (d VirtioRngDevice) KArgs() []string {
	return nil
}

type QOMInteger struct {
	Path     string `json:"path"`
	Property string `json:"property"`
	Value    int    `json:"value"`
}

type TestVM struct {
	cleanup func()
	m       qmp.Monitor
	qemu.VM
}

func NewTestVM(vm *qemu.VM, monsock string, cleanup func()) (*TestVM, error) {
	mon, err := qmp.NewSocketMonitor("unix", monsock, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("Failed to attach to QMP monitor: %v", err)
	}

	if err := mon.Connect(); err != nil {
		return nil, err
	}

	return &TestVM{cleanup, mon, *vm}, nil
}

func (v TestVM) Close() {
	v.m.Disconnect()
	v.cleanup()
}

func (v TestVM) SetQOMInteger(path, property string, value int) error {
	cmd := qmp.Command{
		Execute: "qom-set",
		Args: QOMInteger{
			Path:     path,
			Property: property,
			Value:    value,
		},
	}
	d, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	_, err = v.m.Run(d)
	return err
}
