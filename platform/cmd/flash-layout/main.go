// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

var (
	extra = flag.Int("extra", 0, "Extra amount of bytes to add to the size calculation")
)

func main() {
	flag.Parse()

	size := int64(*extra)
	for _, a := range flag.Args() {
		s, err := os.Stat(a)
		if err != nil {
			log.Fatal(err)
		}
		size += s.Size()
	}

	// Align the split at a 64KiB boundary
	split := (size & ^(1<<16 - 1)) + 64*1024

	// TODO(bluecmd): This could probably be a static file with defines generated
	// by boot-config instead.
	fmt.Printf(`
// AUTOGENERATED BY flash-layout
// SIZE=%d
partitions {
	compatible = "fixed-partitions";
	#address-cells = <1>;
	#size-cells = <1>;

	u-boot@0 {
		reg = <0x0 0x%08x>;
		label = "u-boot";
	};

	ubi@%x {
		reg = <0x%08x 0x0>;
		label = "ubi";
	};
};
`, split, split, split, split)
}
