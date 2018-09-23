// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"

	"github.com/u-root/u-bmc/pkg/ast2400"
)

var (
	bus = flag.Int("bus", 0, "Which I2C bus to watch")

	offset = []uintptr{
		0x40, 0x80, 0xc0, 0x100, 0x140, 0x180, 0x1c0,
		0x300, 0x340, 0x380, 0x3c0, 0x400, 0x440, 0x480,
	}

	state = map[uint32]string{
		0x0: "IDLE",
		0x8: "MACTIVE",
		0x9: "MSTART",
		0xa: "MSTARTR",
		0xb: "MSTOP",
		0xc: "MTXD",
		0xd: "MRXACK",
		0xe: "MRXD",
		0xf: "MTXACK",
		0x1: "SWAIT",
		0x4: "SRXD",
		0x5: "STXACK",
		0x6: "STXD",
		0x7: "SRXACK",
		0x3: "RECOVER",
	}
)

func main() {
	flag.Parse()
	a := ast2400.Open()
	defer a.Close()

	base := uintptr(0x1E78A000)
	active := make([]int, 0)
	for i := 0; i < len(offset); i++ {
		s := a.Mem().MustRead32(base+offset[i]) & 0x3
		clk := a.Mem().MustRead32(base+offset[i]+0x4) & 0xf
		mr := a.Mem().MustRead32(base+offset[i]+0x14) >> 6 & 0x3
		fmt.Printf("I2C bus %d, clk %dx, data buffer %d: ", i, 1<<clk, mr)
		if s&0x1 > 0 {
			fmt.Printf("master ")
			active = append(active, i)
		}
		if s&0x2 > 0 {
			fmt.Printf("slave ")
		}
		fmt.Printf("\n")
	}

	// Slow down I2C for us to dump it
	cr := a.Mem().MustRead32(base + offset[*bus] + 0x4)
	cr = cr | 0xf | 0x3<<8
	a.Mem().MustWrite32(base+offset[*bus]+0x4, cr)

	var pst uint32
	var ptx uint32
	var prx uint32
	var write bool
	txs := make([]byte, 0)
	rxs := make([]byte, 0)
	skip_tx := false
	for {
		rxtx := a.Mem().MustRead32(base+offset[*bus]+0x20) & 0xffff
		rx := rxtx >> 8
		tx := rxtx & 0xff
		mr := a.Mem().MustRead32(base + offset[*bus] + 0x14)
		st := (mr >> 19) & 0xf
		if st == 0 || (pst == st && ptx == tx && prx == rx) {
			continue
		}
		if st == 0x9 || st == 0xa {
			if len(txs) == 0 {
				fmt.Printf("Malformed transaction, skipping\n")
			} else {
				fmt.Printf("End of transaction. Write? %v\n", write)
				addr := txs[0]
				fmt.Printf("Address: %02x\n", addr&0xfe)
				fmt.Printf("TX: ")
				for _, c := range txs[1:] {
					fmt.Printf("%02x ", c)
				}
				fmt.Printf("\nRX: ")
				for _, c := range rxs {
					fmt.Printf("%02x ", c)
				}
				fmt.Printf("\n")
			}
			txs = make([]byte, 0)
			rxs = make([]byte, 0)
			write = tx&0x1 == 0
			skip_tx = true
		} else if len(txs) == 0 && st == 0xc {
			txs = append(txs, byte(tx))
			skip_tx = true
		}

		if st == 0xf {
			rxs = append(rxs, byte(rx))
		}
		if st == 0xd {
			if skip_tx {
				skip_tx = false
			} else {
				txs = append(txs, byte(tx))
			}
		}

		pst = st
		ptx = tx
		prx = rx
	}
}
