// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/u-root/u-bmc/pkg/ast2400"
)

func main() {
	a := ast2400.Open()
	defer a.Close()

	log.SetOutput(os.Stdout)

	// Dump multifunction pin setup
	fmt.Printf("SCU80: %08x\n", a.Mem().MustRead32(ast2400.SCU_BASE+0x80))
	fmt.Printf("SCU84: %08x\n", a.Mem().MustRead32(ast2400.SCU_BASE+0x84))
	fmt.Printf("SCU88: %08x\n", a.Mem().MustRead32(ast2400.SCU_BASE+0x88))
	fmt.Printf("SCU8C: %08x\n", a.Mem().MustRead32(ast2400.SCU_BASE+0x8C))
	fmt.Printf("SCU90: %08x\n", a.Mem().MustRead32(ast2400.SCU_BASE+0x90))
	fmt.Printf("SCU94: %08x\n", a.Mem().MustRead32(ast2400.SCU_BASE+0x94))

	p := a.SnapshotGpio()
	p.Print()
	for {
		s := a.SnapshotGpio()

		if !p.Equals(s) {
			s.Diff(p)
		}
		p = s
		time.Sleep(10 * time.Millisecond)
	}
}
