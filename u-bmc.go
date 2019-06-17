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
	platform   = flag.String("p", "", "Platform to target.")
	packages   = []string{
		"github.com/u-root/u-bmc/cmd/fan",
		"github.com/u-root/u-bmc/cmd/i2cwatcher",
		"github.com/u-root/u-bmc/cmd/login",
		"github.com/u-root/u-bmc/cmd/socreset",
		"github.com/u-root/u-bmc/cmd/ubmcctl",
		"github.com/u-root/u-root/cmds/core/*",
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

	builder := builder.BBBuilder{}
	archiver, err := initramfs.GetArchiver("cpio")
	if err != nil {
		return err
	}

	tempDir, err := ioutil.TempDir("", "u-root")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	logger := log.New(os.Stderr, "", log.LstdFlags)
	w, err := archiver.OpenWriter(logger, *outputPath, env.GOOS, env.GOARCH)
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
		BaseArchive:  baseFile,
		TempDir:      tempDir,
		OutputFile:   w,
		InitCmd:      "init",
		DefaultShell: "elvish",
	}
	return uroot.CreateInitramfs(logger, opts)
}
