// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

const (
	// Serial output is written to this directory and picked up by circleci, or
	// you, if you want to read the serial logs. TODO(MDr164): Reimplement
	//logDir = "serial"
	// Allow 30 seconds * TimeoutMultiplier (2.0 right now) timeout for tests
	defaultTimeout = 60 * time.Second
)

func BMCTest(t *testing.T, bit int, testcmd string) (*TestVM, func()) {
	buildImageAndInitrd(t, testcmd)
	cleanup := func() { dirCleanup(t) }
	tvm, err := NewTestVM(t, bit, defaultTimeout, cleanup)
	if err != nil {
		dirCleanup(t)
		t.Fatalf("failed to create test VM: %v", err)
	}

	return tvm, tvm.Close
}

func buildImageAndInitrd(t *testing.T, testcmd string) {
	target, err := os.Create("../TARGET")
	if err != nil {
		t.Fatal("could not create TARGET file")
	}
	_, err = target.Write([]byte("qemu-virt-a72"))
	if err != nil {
		t.Fatal("could not set TARGET")
	}
	target.Close()

	taskfile, err := filepath.Abs("../Taskfile.yml")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("task", "-t", taskfile, "build", "--", testcmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = filepath.Dir(taskfile)
	err = cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
}

func dirCleanup(t *testing.T) {
	if t.Failed() {
		t.Log("keeping build files")
	} else {
		taskfile, err := filepath.Abs("../Taskfile.yml")
		if err != nil {
			t.Log(err)
		}
		cmd := exec.Command("task", "-t", taskfile, "clean")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = filepath.Dir(taskfile)
		err = cmd.Run()
		if err != nil {
			t.Log("failed to remove build artifacts")
		}
	}
}
