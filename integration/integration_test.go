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

	"github.com/u-root/u-root/pkg/cp"
	"github.com/u-root/u-root/pkg/golang"
	"github.com/u-root/u-root/pkg/qemu"
	"github.com/u-root/u-root/pkg/uroot"
	"github.com/u-root/u-root/pkg/uroot/builder"
	"github.com/u-root/u-root/pkg/uroot/initramfs"
)

const (
	ubootImage = "../u-boot/u-boot-512.bin"
	bootImage  = "../boot.ubifs.img"
	// Serial output is written to this directory and picked up by circleci, or
	// you, if you want to read the serial logs.
	logDir = "serial"
)

func init() {
	// Allow 30 seconds * TimeoutMultiplier (2.0 right now) timeout for tests
	qemu.DefaultTimeout = 30 * time.Second
}

// Returns temporary directory and QEMU instance.
//
// - `uinitName` is the name of a directory containing uinit found at
//   `github.com/u-root/u-bmc/integration/testdata`.
func testWithQEMU(t *testing.T, uinitName string, logName string, extraEnv []string) (string, *qemu.QEMU) {
	if _, ok := os.LookupEnv("UROOT_QEMU"); !ok {
		t.Skip("test is skipped unless UROOT_QEMU is set")
	}
	if _, err := os.Stat(ubootImage); err != nil {
		t.Fatalf("u-boot not built, cannot test")
	}
	if _, err := os.Stat(bootImage); err != nil {
		t.Fatalf("boot partition not built, cannot test")
	}

	// TempDir
	tmpDir, err := ioutil.TempDir("", "ubmc-integration")
	if err != nil {
		t.Fatal(err)
	}

	// Env
	env := golang.Default()
	env.CgoEnabled = false
	env.GOARCH = "arm"

	// OutputFile
	outputFile := filepath.Join(tmpDir, "initramfs.cpio")
	w, err := initramfs.CPIO.OpenWriter(outputFile, "", "")
	if err != nil {
		t.Fatal(err)
	}

	// Copy build artifacts to our temp dir
	if err := os.Mkdir(filepath.Join(tmpDir, "u-boot"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := cp.Copy("../u-boot/u-boot-512.bin", filepath.Join(tmpDir, "u-boot", "u-boot-512.bin")); err != nil {
		t.Fatal(err)
	}
	if err := cp.Copy("../boot.ubifs.img", filepath.Join(tmpDir, "boot.ubifs.img")); err != nil {
		t.Fatal(err)
	}
	if err := cp.Copy("../ubi.cfg", filepath.Join(tmpDir, "ubi.cfg")); err != nil {
		t.Fatal(err)
	}

	// Build u-root
	opts := uroot.Opts{
		Env: env,
		Commands: []uroot.Commands{
			{
				Builder: builder.BusyBox,
				Packages: []string{
					"github.com/u-root/u-root/cmds/*",
					path.Join("github.com/u-root/u-bmc/integration/testdata", uinitName, "uinit"),
				},
			},
		},
		TempDir:      tmpDir,
		BaseArchive:  uroot.DefaultRamfs.Reader(),
		OutputFile:   w,
		InitCmd:      "init",
		DefaultShell: "elvish",
	}
	logger := log.New(os.Stderr, "", log.LstdFlags)
	if err := uroot.CreateInitramfs(logger, opts); err != nil {
		t.Fatal(err)
	}

	makefile, err := filepath.Abs("../Makefile")
	if err != nil {
		t.Fatal(err)
	}

	build(t, tmpDir, makefile, extraEnv)

	flash := filepath.Join(tmpDir, "flash.img")
	extraArgs := []string{
		"-drive", "file=" + flash + ",format=raw,if=mtd",
		"-M", "palmetto-bmc",
		"-m", "256",
	}

	// Create file for serial logs.
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("could not create serial log directory: %v", err)
	}
	logFile, err := os.Create(path.Join(logDir, logName+".log"))
	if err != nil {
		t.Fatalf("could not create log file: %v", err)
	}

	// Start QEMU
	q := &qemu.QEMU{
		ExtraArgs:    extraArgs,
		SerialOutput: logFile,
	}
	t.Logf("command line:\n%s", q.CmdLineQuoted())
	if err := q.Start(); err != nil {
		t.Fatal("could not spawn QEMU: ", err)
	}
	return tmpDir, q
}

func build(t *testing.T, tmpDir string, makefile string, extraEnv []string) {
	cmd := exec.Command(
		"make", "-f", makefile, "flash.img",
		"-o", "u-boot/u-boot-512.bin",
		"-o", "boot/signer/signer",
		"-o", "boot.ubifs.img",
		"-o", "initramfs.cpio")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), extraEnv...)
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
}

func cleanup(t *testing.T, tmpDir string, q *qemu.QEMU) {
	q.Close()
	if t.Failed() {
		t.Log("keeping temp dir: ", tmpDir)
	} else {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("failed to remove temporary directory %s", tmpDir)
		}
	}
}
