// Copyright 2016-2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// mounter mounts the root file system on /mnt/ and then
// switches root into the mountpoint. This programm
// allows for more complex rootfs setups.

package main

import (
	"flag"
	"log"
	"os"

	"github.com/u-root/u-root/pkg/mount"
	"golang.org/x/sys/unix"
)

const UUID = "26ab0401-3f49-4fc2-a172-c8aa02aceaf3"

var (
	mtd = flag.Bool("mtd", false, "Mount and load u-bmc from MTD flash")
	blk = flag.Bool("blk", false, "Mount and load u-bmc from block device")
)

func main() {
	flag.Parse()
	if *mtd && *blk {
		log.Fatal("Please choose either mtd or blk, not both!")
	}
	if !*mtd && !*blk {
		log.Fatal("Please choose either mtd or blk!")
	}

	dirs := []string{"/mnt", "/ro", "/tmp", "/proc", "/sys"}
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			log.Fatalf("Mkdir(%s): %v", dir, err)
		}
	}

	if *mtd {
		err := unix.Mount("ubi0:root", "/ro", "ubifs", unix.MS_RDONLY, "")
		if err != nil {
			log.Fatalf("Mount(ubi0:root): %v", err)
		}
	}
	if *blk {
		err := unix.Mount("UUID="+UUID, "/ro", "erofs", unix.MS_RDONLY, "")
		if err != nil {
			log.Fatalf("Mount(%s): %v", "UUID="+UUID, err)
		}
	}
	err := unix.Mount("tmpfs", "/tmp", "tmpfs", 0, "")
	if err != nil {
		log.Fatalf("Mount(tmpfs): %v", err)
	}
	tmpdirs := []string{"/tmp/upper", "/tmp/work"}
	for _, dir := range tmpdirs {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			log.Fatalf("Mkdir(%s): %v", dir, err)
		}
	}
	err = unix.Mount("overlayfs", "/mnt", "overlay", 0, "lowerdir=/ro,upperdir=/tmp/upper,workdir=/tmp/work")
	if err != nil {
		log.Fatalf("Mount(overlayfs): %v", err)
	}
	err = unix.Mount("proc", "/proc", "proc", 0, "")
	if err != nil {
		log.Fatalf("Mount(proc): %v", err)
	}
	err = unix.Mount("sysfs", "/sys", "sysfs", 0, "")
	if err != nil {
		log.Fatalf("Mount(sysfs): %v", err)
	}

	err = mount.SwitchRoot("/mnt", "/bin/init")
	if err != nil {
		log.Fatalf("SwitchRoot: %v", err)
	}
}
