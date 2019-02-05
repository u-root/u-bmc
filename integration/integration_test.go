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
	"regexp"
	"strings"
	"testing"
	"time"

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
	qemu.DefaultTimeout = 30 * time.Second
}

// MakefileVars holds variables specified in u-bmc Makefile.
type MakefileVars map[string]string

// ReadMakefile reads a map of variables from the Makefile.
func ReadMakefile() (MakefileVars, error) {
	cmd := exec.Command("make", "-C", "..", "vars")
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`([^=\n]+)=([^\n]+)`)
	m := MakefileVars{}
	for _, v := range re.FindAllStringSubmatch(string(out), -1) {
		m[v[1]] = v[2]
	}
	return m, nil
}

// Returns temporary directory and QEMU instance.
//
// - `uinitName` is the name of a directory containing uinit found at
//   `github.com/u-root/u-bmc/integration/testcmd`.
func testWithQEMU(t *testing.T, uinitName string, logName string, extraEnv []string) (string, *qemu.QEMU) {
	if _, ok := os.LookupEnv("UROOT_QEMU"); !ok {
		t.Skip("test is skipped unless UROOT_QEMU is set")
	}
	if _, err := os.Stat("../boot/boot.bin"); err != nil {
		t.Fatalf("u-bmc not built, cannot test")
	}

	makeVars, err := ReadMakefile()
	if err != nil {
		t.Fatalf("unable to read Makefile: %v", err)
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

	// Build u-root
	opts := uroot.Opts{
		Env: env,
		Commands: []uroot.Commands{
			{
				Builder: builder.BusyBox,
				Packages: []string{
					"github.com/u-root/u-root/cmds/init",
					path.Join("github.com/u-root/u-bmc/integration/testcmd", uinitName, "uinit"),
				},
			},
		},
		TempDir:     tmpDir,
		BaseArchive: uroot.DefaultRamfs.Reader(),
		OutputFile:  w,
		InitCmd:     "init",
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

	flash := filepath.Join(tmpDir, "flash.sim.img")
	flags := makeVars["QEMUFLAGS"]
	flags = strings.Replace(flags, "flash.sim.img", flash, 1)
	extraArgs := strings.Fields(flags)

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
		"make", "-f", makefile, "flash.sim.img", "-o", "initramfs.cpio")
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
