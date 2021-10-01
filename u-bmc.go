// Copyright 2015-2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/u-root/gobusybox/src/pkg/bb"
	"github.com/u-root/gobusybox/src/pkg/golang"
)

// Flags for the gobusybox builder
var (
	// Using this explicit list we can easier restrict which commands
	// end up in our busybox binary
	urootWhitelist = []string{
		"backoff",
		"basename",
		"bind",
		"blkid",
		"cat",
		"chmod",
		"cmp",
		"comm",
		"cp",
		"date",
		"dd",
		"df",
		"dirname",
		"dmesg",
		"echo",
		"elvish", //TODO(MDr164) maybe use upstream instead?
		"false",
		"find",
		"free",
		"grep",
		"hexdump",
		"hostname",
		"hwclock",
		"id",
		"init", //TODO(MDr164) write own slimmed down u-bmc init
		"io",
		"ip",
		"kill",
		"ln",
		"losetup",
		"ls",
		"lsmod",
		"man",
		"md5sum", //TODO(MDr164) should be obsolete
		"mkdir",
		"mkfifo",
		"mknod",
		"mktemp",
		"more",
		"mount",
		"mv",
		"netcat",
		"pci",
		"printenv",
		"ps",
		"pwd",
		"readlink",
		"rm",
		"scp",
		"seq",
		"shasum",
		"shutdown",
		"sleep",
		"sort",
		"sshd", //TODO(MDr164) replace with in-process sshd
		"strace",
		"strings",
		"stty",
		"sync",
		"tail",
		"tar",
		"tee",
		"time",
		"tr",
		"true",
		"truncate",
		"ts",
		"umount",
		"unshare", //TODO(MDr164) probably not needed
		"uptime",
		"watchdog",
		"watchdogd",
		"wc",
		"wget",
		"which",
		"yes",
	}
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
	urootcmds := urootAbsEach(urootWhitelist)
	bmccmds, err := filepath.Glob("../cmd/*")
	if err != nil {
		return err
	}
	var extracmds []string
	for _, arg := range flag.Args() {
		path, err := filepath.Glob(arg)
		if err != nil {
			continue
		}
		extracmds = append(extracmds, path...)
	}
	var commands []string
	commands = append(commands, bmccmds...)
	commands = append(commands, urootcmds...)
	commands = append(commands, extracmds...)
	commands = removeDuplicate(commands)
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

func urootAbsEach(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i], _ = filepath.Abs("../../u-root/cmds/core/" + s)
	}
	return out
}

func removeDuplicate(in []string) []string {
	base := make(map[string]int)
	for i, s := range in {
		base[filepath.Base(s)] = i
	}
	unique := make([]string, len(in))
	for _, i := range base {
		unique[i] = in[i]
	}
	var out []string
	for _, s := range unique {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
