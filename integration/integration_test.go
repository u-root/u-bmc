// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package integration

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"testing"
	"time"

	urootint "github.com/u-root/u-root/integration"
	"github.com/u-root/u-root/pkg/golang"
	"github.com/u-root/u-root/pkg/qemu"
	"github.com/u-root/u-root/pkg/uroot"
	"github.com/u-root/u-root/pkg/uroot/builder"
	"github.com/u-root/u-root/pkg/uroot/initramfs"
)

const (
	// Serial output is written to this directory and picked up by circleci, or
	// you, if you want to read the serial logs.
	logDir = "serial"
)

func init() {
	// Allow 30 seconds * TimeoutMultiplier (2.0 right now) timeout for tests
	qemu.DefaultTimeout = 60 * time.Second
}

type Options urootint.Options

func BMCTest(t *testing.T, o *Options) (*TestVM, func()) {
	if _, ok := os.LookupEnv("UBMC_QEMU"); !ok {
		t.Skip("test is skipped unless UBMC_QEMU is set")
	}
	if _, err := os.Stat("../boot/boot.bin"); err != nil {
		t.Fatalf("u-bmc not built, cannot test")
	}

	// TempDir
	tmpDir, err := ioutil.TempDir("", "ubmc-integration")
	if err != nil {
		t.Fatal(err)
	}

	// Env
	if o.Env == nil {
		env := golang.Default()
		env.CgoEnabled = false
		env.GOARCH = "arm"
		o.Env = &env
	}

	_ = buildInitramfs(t, tmpDir, o)
	flash := buildFlash(t, tmpDir, o)
	monsock := filepath.Join(tmpDir, "qmp.sock")
	q := &qemu.Options{
		QEMUPath: os.Getenv("UBMC_QEMU"),
		// TODO(bluecmd): Right now only one platform is supported for tests
		Devices: []qemu.Device{
			MachineDevice{"palmetto-bmc"},
			MemoryDevice{128},
			FlashDevice{flash},
			QemuMonitorDevice{monsock},
		},
	}
	vm, vmCleanup := qemuTest(t, q, o)
	cleanup := func() {
		vmCleanup()
		dirCleanup(t, tmpDir)
	}
	// Wait for monitor socket to show up
	for _, i := range []int{100, 500, 1000, 5000, -1} {
		if _, err := os.Stat(monsock); err == nil {
			break
		}
		if i == -1 {
			t.Fatalf("Timed out waiting for monitor socket to appear")
		}
		time.Sleep(time.Duration(i) * time.Millisecond)
	}
	tvm, err := NewTestVM(vm, monsock, cleanup)
	if err != nil {
		cleanup()
		t.Fatalf("Failed to create test VM: %v", err)
	}

	return tvm, tvm.Close
}

func NativeTest(t *testing.T, o *Options) (*qemu.VM, func()) {
	if _, ok := os.LookupEnv("UBMC_NATIVE_QEMU"); !ok {
		t.Skip("test is skipped unless UBMC_NATIVE_QEMU is set")
	}
	kernel, err := filepath.Abs("bzImage")
	if err != nil {
		t.Skip("test is skipped unless bzImage is built")
	}

	// TempDir
	tmpDir, err := ioutil.TempDir("", "ubmc-integration")
	if err != nil {
		t.Fatal(err)
	}

	// Env
	if o.Env == nil {
		env := golang.Default()
		env.CgoEnabled = false
		o.Env = &env
	}

	i := buildInitramfs(t, tmpDir, o)
	q := &qemu.Options{
		Initramfs: i,
		Kernel:    kernel,
		QEMUPath:  os.Getenv("UBMC_NATIVE_QEMU"),
		Devices: []qemu.Device{
			VirtioRngDevice{},
		},
	}
	vm, vmCleanup := qemuTest(t, q, o)

	return vm, func() {
		vmCleanup()
		dirCleanup(t, tmpDir)
	}
}

func buildInitramfs(t *testing.T, tmpDir string, o *Options) string {
	// OutputFile
	logger := log.New(os.Stderr, "", log.LstdFlags)
	outputFile := filepath.Join(tmpDir, "initramfs.cpio")
	w, err := initramfs.CPIO.OpenWriter(logger, outputFile, "", "")
	if err != nil {
		t.Fatal(err)
	}

	// Build u-root
	opts := uroot.Opts{
		Env: *o.Env,
		Commands: []uroot.Commands{
			{
				Builder:  builder.BusyBox,
				Packages: o.Cmds,
			},
		},
		TempDir:     tmpDir,
		BaseArchive: uroot.DefaultRamfs.Reader(),
		OutputFile:  w,
		InitCmd:     "init",
	}
	if err := uroot.CreateInitramfs(logger, opts); err != nil {
		t.Fatal(err)
	}
	return outputFile
}

func qemuTest(t *testing.T, q *qemu.Options, o *Options) (*qemu.VM, func()) {
	// Create file for serial logs.
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("could not create serial log directory: %v", err)
	}
	logFile, err := os.Create(path.Join(logDir, o.Name+".log"))
	if err != nil {
		t.Fatalf("could not create log file: %v", err)
	}

	q.SerialOutput = logFile
	if q.Devices != nil {
		q.Devices = append(q.Devices, o.Network)
	} else {
		q.Devices = []qemu.Device{o.Network}
	}

	vm, err := q.Start()
	if err != nil {
		t.Fatalf("Failed to start QEMU VM %s: %v", o.Name, err)
	}
	t.Logf("QEMU command line for %s:\n%s", o.Name, vm.CmdlineQuoted())
	return vm, func() {
		vm.Close()
	}
}

func buildFlash(t *testing.T, tmpDir string, o *Options) string {
	makefile, err := filepath.Abs("../Makefile")
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(
		"make", "-f", makefile, "flash.sim.img", "-o", "initramfs.cpio")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), o.ExtraBuildEnv...)
	err = cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(tmpDir, "flash.sim.img")
}

func dirCleanup(t *testing.T, tmpDir string) {
	if t.Failed() {
		t.Log("keeping temp dir: ", tmpDir)
	} else {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("failed to remove temporary directory %s", tmpDir)
		}
	}
}
