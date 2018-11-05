// Copyright 2015-2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/u-root/u-root/pkg/golang"
	"github.com/u-root/u-root/pkg/uroot"
	"github.com/u-root/u-root/pkg/uroot/builder"
	"github.com/u-root/u-root/pkg/uroot/initramfs"
)

// Flags for u-root builder.
var (
	outputPath = flag.String("o", "", "Path to output initramfs file.")
	platform = flag.String("p", "", "Platform to target.")
	packages = []string{
		"github.com/u-root/u-bmc/cmd/fan",
		"github.com/u-root/u-bmc/cmd/i2cwatcher",
		"github.com/u-root/u-bmc/cmd/socreset",
		"github.com/u-root/u-bmc/cmd/ubmcctl",
		// Based on core in u-root
		"github.com/u-root/u-root/cmds/ansi",
		"github.com/u-root/u-root/cmds/boot",
		"github.com/u-root/u-root/cmds/cat",
		"github.com/u-root/u-root/cmds/cbmem",
		"github.com/u-root/u-root/cmds/chmod",
		"github.com/u-root/u-root/cmds/chroot",
		"github.com/u-root/u-root/cmds/cmp",
		"github.com/u-root/u-root/cmds/console",
		"github.com/u-root/u-root/cmds/cp",
		"github.com/u-root/u-root/cmds/cpio",
		"github.com/u-root/u-root/cmds/date",
		"github.com/u-root/u-root/cmds/dd",
		"github.com/u-root/u-root/cmds/df",
		"github.com/u-root/u-root/cmds/dhclient",
		"github.com/u-root/u-root/cmds/dirname",
		"github.com/u-root/u-root/cmds/dmesg",
		"github.com/u-root/u-root/cmds/echo",
		"github.com/u-root/u-root/cmds/elvish",
		"github.com/u-root/u-root/cmds/false",
		"github.com/u-root/u-root/cmds/field",
		"github.com/u-root/u-root/cmds/find",
		"github.com/u-root/u-root/cmds/free",
		"github.com/u-root/u-root/cmds/freq",
		"github.com/u-root/u-root/cmds/gpgv",
		"github.com/u-root/u-root/cmds/gpt",
		"github.com/u-root/u-root/cmds/grep",
		"github.com/u-root/u-root/cmds/gzip",
		"github.com/u-root/u-root/cmds/hexdump",
		"github.com/u-root/u-root/cmds/hostname",
		"github.com/u-root/u-root/cmds/id",
		"github.com/u-root/u-root/cmds/init",
		"github.com/u-root/u-root/cmds/insmod",
		"github.com/u-root/u-root/cmds/installcommand",
		"github.com/u-root/u-root/cmds/io",
		"github.com/u-root/u-root/cmds/ip",
		"github.com/u-root/u-root/cmds/kexec",
		"github.com/u-root/u-root/cmds/kill",
		"github.com/u-root/u-root/cmds/lddfiles",
		"github.com/u-root/u-root/cmds/ln",
		"github.com/u-root/u-root/cmds/losetup",
		"github.com/u-root/u-root/cmds/ls",
		"github.com/u-root/u-root/cmds/lsmod",
		"github.com/u-root/u-root/cmds/mkdir",
		"github.com/u-root/u-root/cmds/mkfifo",
		"github.com/u-root/u-root/cmds/mknod",
		"github.com/u-root/u-root/cmds/modprobe",
		"github.com/u-root/u-root/cmds/mount",
		"github.com/u-root/u-root/cmds/msr",
		"github.com/u-root/u-root/cmds/mv",
		"github.com/u-root/u-root/cmds/netcat",
		"github.com/u-root/u-root/cmds/ntpdate",
		"github.com/u-root/u-root/cmds/pci",
		"github.com/u-root/u-root/cmds/ping",
		"github.com/u-root/u-root/cmds/printenv",
		"github.com/u-root/u-root/cmds/ps",
		"github.com/u-root/u-root/cmds/pwd",
		"github.com/u-root/u-root/cmds/pxeboot",
		"github.com/u-root/u-root/cmds/readlink",
		"github.com/u-root/u-root/cmds/rm",
		"github.com/u-root/u-root/cmds/rmmod",
		"github.com/u-root/u-root/cmds/rsdp",
		"github.com/u-root/u-root/cmds/scp",
		"github.com/u-root/u-root/cmds/seq",
		"github.com/u-root/u-root/cmds/shutdown",
		"github.com/u-root/u-root/cmds/sleep",
		"github.com/u-root/u-root/cmds/sort",
		"github.com/u-root/u-root/cmds/sshd",
		"github.com/u-root/u-root/cmds/stty",
		"github.com/u-root/u-root/cmds/switch_root",
		"github.com/u-root/u-root/cmds/sync",
		"github.com/u-root/u-root/cmds/tail",
		"github.com/u-root/u-root/cmds/tee",
		"github.com/u-root/u-root/cmds/true",
		"github.com/u-root/u-root/cmds/truncate",
		"github.com/u-root/u-root/cmds/umount",
		"github.com/u-root/u-root/cmds/uname",
		"github.com/u-root/u-root/cmds/uniq",
		"github.com/u-root/u-root/cmds/unshare",
		"github.com/u-root/u-root/cmds/validate",
		"github.com/u-root/u-root/cmds/vboot",
		"github.com/u-root/u-root/cmds/wc",
		"github.com/u-root/u-root/cmds/wget",
		"github.com/u-root/u-root/cmds/which",
	}
)

func main() {
	flag.Parse()

	// Main is in a separate functions so defers run on return.
	if err := Main(); err != nil {
		log.Fatal(err)
	}
	log.Printf("Successfully wrote initramfs.")
}

// Main is a separate function so defers are run on return, which they wouldn't
// on exit.
func Main() error {
	env := golang.Default()
	if env.CgoEnabled {
		// TODO(bluecmd): Might need CGO for pcap if that should be included
		// Given that we already depend on a gcc being available for u-boot and
		// the linux kernel, this might be fine. Especially if we need to do the
		// yocto route down the line.
		log.Printf("Disabling CGO for u-bmc...")
		env.CgoEnabled = false
	}
	log.Printf("Build environment: %s", env)
	if env.GOOS != "linux" {
		log.Printf("GOOS is not linux. Did you mean to set GOOS=linux?")
	}

	builder, err := builder.BBBuilder{}
	archiver, err := initramfs.GetArchiver("cpio")
	if err != nil {
		return err
	}

	tempDir, err := ioutil.TempDir("", "u-root")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	w, err := archiver.OpenWriter(*outputPath, env.GOOS, env.GOARCH)
	if err != nil {
		return err
	}

	pkgs := append(packages, fmt.Sprintf("github.com/u-root/u-bmc/platform/%s/cmd/*", *platform))

	baseFile := uroot.DefaultRamfs.Reader()

	opts := uroot.Opts{
		Env: env,
		Commands: []uroot.Commands{
			{
				Builder:  builder,
				Packages: pkgs,
			},
		},
		BaseArchive:     baseFile,
		TempDir:         tempDir,
		OutputFile:      w,
		InitCmd:         "init",
		DefaultShell:    "elvish",
	}
	logger := log.New(os.Stderr, "", log.LstdFlags)
	return uroot.CreateInitramfs(logger, opts)
}
