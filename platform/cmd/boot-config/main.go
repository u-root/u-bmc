// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
)

var (
	ramStartStr = flag.String("ram-start", "", "Address where the RAM starts")
	ramSizeStr  = flag.String("ram-size", "", "Size of the RAM")
	initrdPath  = flag.String("initrd", "", "Initrd file")
	dtbPath     = flag.String("dtb", "", "DTB file")
)

func main() {
	flag.Parse()

	dtb, err := os.Stat(*dtbPath)
	if err != nil {
		log.Fatalf("os.Stat(%s): %v", *dtbPath, err)
	}

	initrd, err := os.Stat(*initrdPath)
	if err != nil {
		log.Fatalf("os.Stat(%s): %v", *initrdPath, err)
	}

	ramStart, err := strconv.ParseInt(*ramStartStr, 0, 64)
	if err != nil {
		log.Fatalf("strconv.ParseInt(%s): %v", *ramStartStr, err)
	}
	ramSize, err := strconv.ParseInt(*ramSizeStr, 0, 64)
	if err != nil {
		log.Fatalf("strconv.ParseInt(%s): %v", *ramSizeStr, err)
	}
	ramEnd := ramStart + ramSize

	dtbStart := (ramEnd - dtb.Size()) & ^0x3
	// Align to 64 KiB to make the kernel happy
	initrdEnd := (dtbStart - 64*1024)
	initrdStart := (initrdEnd - initrd.Size()) & ^(64*1024-1)
	initrdEnd = initrdStart + initrd.Size()

	fmt.Printf(`
#define CONFIG_INITRD_START 0x%x
#define CONFIG_INITRD_END   0x%x
#define CONFIG_DTB_START    0x%x
#define CONFIG_DTB_END      CONFIG_RAM_END
#define CONFIG_RAM_START    0x%x
#define CONFIG_RAM_SIZE     0x%x
#define CONFIG_RAM_END      0x%x
`, initrdStart, initrdEnd, dtbStart, ramStart, ramSize, ramEnd)
}
