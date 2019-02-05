// Copyright 2018-2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aspeed

import (
	"flag"
	"fmt"
	"os"
	"time"
)

var (
	printLpcStats = flag.Bool("print_lpc_stats", false, "At the end of the run, print LPC statistics")
	// TODO(bluecmd): Maybe it's worth only caching address?
	// There has been some weird lockups on doing data caching,
	// but address caching seems fine.
	lpcCache = flag.Bool("lpc_cache", false, "Do not write values that match cached view of LPC2AHB F0-9 registers")
)

type lpc struct {
	p   *os.File
	off int64
	// Fx register cache (F0-F8)
	f [9]byte

	// debugging stats
	stat struct {
		wr_time         time.Duration
		rd_time         time.Duration
		wr_count        int
		rd_count        int
		cached_wr_count int
	}
}

func openLpcMemory(port int) *lpc {
	p, err := os.OpenFile("/dev/port", os.O_RDWR, 0600)
	if err != nil {
		panic(err)
	}

	l := &lpc{p: p, off: int64(port)}
	l.unlock()
	l.selectDevice(0xd)

	// "cache invalidation" by making the controller match our cache
	for i := 0; i < 9; i++ {
		l.ctrl(byte(0xf0 + i))
		l.w(byte(0))
	}

	l.enable()
	return l
}

func (l *lpc) ctrl(d byte) {
	b := []byte{d}
	l.stat.wr_count++
	t := time.Now()
	l.p.WriteAt(b, l.off)
	l.stat.wr_time = l.stat.wr_time + time.Now().Sub(t)
}

func (l *lpc) wf(f int, d byte) {
	// Write F0-8 reg through cache to avoid redundant writes
	i := f - 0xf0
	if l.f[i] != d || !*lpcCache {
		l.ctrl(byte(f))
		l.w(d)
		l.f[i] = d
	} else {
		l.stat.cached_wr_count++
	}
}

func (l *lpc) w(d byte) {
	b := []byte{d}
	l.stat.wr_count++
	t := time.Now()
	l.p.WriteAt(b, l.off+1)
	l.stat.wr_time = l.stat.wr_time + time.Now().Sub(t)
}

func (l *lpc) r() byte {
	b := make([]byte, 1)
	l.stat.rd_count++
	t := time.Now()
	l.p.ReadAt(b, l.off+1)
	l.stat.rd_time = l.stat.rd_time + time.Now().Sub(t)
	return b[0]
}

func (l *lpc) enable() {
	// Enable SIO iLPC2AHB
	// TODO(bluecmd): Does this make sense? If it's not enabled we couldn't
	// enable it, right?
	l.ctrl(0x30)
	l.w(0x1)
}

func (l *lpc) unlock() {
	// Unlock SIO
	l.ctrl(0xa5)
	l.ctrl(0xa5)
}

func (l *lpc) Close() {
	// Lock SIO
	l.ctrl(0xaa)

	// Print some stats
	if *printLpcStats {
		fmt.Printf("LPC stats: %v RDs (time %v), %v WRs (time %v), cached %v WRs\n",
			l.stat.rd_count, l.stat.rd_time, l.stat.wr_count, l.stat.wr_time,
			l.stat.cached_wr_count)
	}
}

func (l *lpc) selectDevice(d int) {
	l.ctrl(0x07)
	l.w(byte(d))
}

func (l *lpc) addr(a uintptr) {
	l.wf(0xf0, byte(a>>24&0xff))
	l.wf(0xf1, byte(a>>16&0xff))
	l.wf(0xf2, byte(a>>8&0xff))
	l.wf(0xf3, byte(a&0xff))
}

func (l *lpc) MustRead32(a uintptr) uint32 {
	l.addr(a)
	// Select 32 bit
	l.wf(0xf8, 0x2)
	// Trigger
	l.ctrl(0xfe)
	l.r()
	// Read 32 bit
	var res uint32
	l.ctrl(0xf4)
	f := l.r()
	l.f[4] = f
	res |= uint32(f) << 24
	l.ctrl(0xf5)
	f = l.r()
	l.f[5] = f
	res |= uint32(f) << 16
	l.ctrl(0xf6)
	f = l.r()
	l.f[6] = f
	res |= uint32(f) << 8
	l.ctrl(0xf7)
	f = l.r()
	l.f[7] = f
	res |= uint32(f)
	return res
}

func (l *lpc) MustRead8(a uintptr) uint8 {
	l.addr(a)
	// Select 8 bit
	l.wf(0xf8, 0)
	// Trigger
	l.ctrl(0xfe)
	l.r()
	// Read 8 bit
	// TODO(bluecmd) WHat about the other regs here?
	l.ctrl(0xf7)
	f := l.r()
	l.f[7] = f
	return f
}

func (l *lpc) MustWrite32(a uintptr, d uint32) {
	l.addr(a)
	// Select 32 bit
	l.wf(0xf8, 0x2)

	// Write 32 bit
	l.wf(0xf4, byte(d>>24&0xff))
	l.wf(0xf5, byte(d>>16&0xff))
	l.wf(0xf6, byte(d>>8&0xff))
	l.wf(0xf7, byte(d&0xff))
	// Trigger
	l.ctrl(0xfe)
	l.w(0xcf)
}

func (l *lpc) MustWrite8(a uintptr, d uint8) {
	l.addr(a)
	// Select 8 bit
	l.wf(0xf8, 0)
	// Write 8 bit
	l.wf(0xf7, byte(d&0xff))
	// Trigger
	l.ctrl(0xfe)
	l.w(0xcf)
}
