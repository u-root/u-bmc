// Copyright 2015-2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/u-root/gobusybox/src/pkg/bb"
	"github.com/u-root/gobusybox/src/pkg/golang"
)

// Flags for the gobusybox builder
var (
	platform = flag.String("plat", "", "Platform to target.")
)

func main() {
	flag.Parse()

	// Main is in a separate functions so defers run on return
	if err := Main(); err != nil {
		log.Fatal(err)
	}
	log.Print("Successfully created root directory")
}

// Main is a separate function so defers are run on return, which they wouldn't
// on exit
func Main() error {
	// Make sure to disable CGO as it's not supported
	env := golang.Default()
	if env.CgoEnabled {
		env.CgoEnabled = false
	}

	// Resolve paths to all individual go commands for the busybox
	bmccmds, err := filepath.Glob("../cmd/*")
	if err != nil {
		return err
	}
	urootcmds, err := filepath.Glob("../../u-root/cmds/core/*")
	if err != nil {
		return err
	}
	platcmds, err := filepath.Glob(fmt.Sprintf("../platform/%s/cmd/*", *platform))
	if err != nil {
		return err
	}
	var commands []string
	commands = append(commands, bmccmds...)
	commands = append(commands, urootcmds...)
	commands = append(commands, platcmds...)
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Define some options for building the busybox
	opts := bb.Opts{
		Env:          env,
		CommandPaths: commands,
		BinaryPath:   pwd + "/rootfs/bin/bb",
		GoBuildOpts: &golang.BuildOpts{
			NoStrip:        false,
			NoTrimPath:     false,
			EnableInlining: false,
		},
		AllowMixedMode: true,
	}

	// Build the bb binary
	err = bb.BuildBusybox(&opts)
	if err != nil {
		return err
	}

	// Create symlinks for the bb binary and all commands
	for _, cmd := range commands {
		os.RemoveAll("rootfs/bin/" + filepath.Base(cmd))
		err := os.Symlink("bb", "rootfs/bin/"+filepath.Base(cmd))
		if err != nil {
			return err
		}
	}
	os.RemoveAll("rootfs/bin/sh")
	err = os.Symlink("elvish", "rootfs/bin/sh")
	if err != nil {
		return err
	}

	// Create default directory structure
	directories := []string{"dev", "proc", "sys", "usr/lib", "var/log", "tmp", "etc", "boot", "config"}
	for _, dir := range directories {
		err := os.MkdirAll("rootfs/"+dir, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}
