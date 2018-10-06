// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ast2400

import (
	"errors"
	"fmt"
	"log"
	"time"
)

const (
	CS0_CTRL         uintptr = 0x1e620010
	CS0_SEGMENT_ADDR uintptr = 0x1e620030
	FLASH_START      uintptr = 0x20000000
	SPI_READ_TIMINGS uintptr = 0x1e620094

	// TODO(bluecmd): Extract this to a non-aspeed package, or see if there
	// already is one out there for Golang.
	MX25L256_ID = 0x1920c2
	MT25Q512_ID = 0x20ba20

	OP_ID                = 0x9f
	OP_READ_STATUS       = 0x05
	COMMON_OP_WREN         = 0x06
	COMMON_OP_BLOCK_ERASE  = 0xd8
	COMMON_OP_PAGE_PROGRAM = 0x02
	COMMON_OP_EN4B         = 0xb7
	COMMON_OP_EX4B         = 0xe9
	COMMON_OP_FAST_READ    = 0x0b
	COMMON_OP_RD_FLAG_REG  = 0x70

	MT25Q_WR_LOCK_BITS      = 0xe5
	MT25Q_RD_LOCK_BITS      = 0xe8
)

var (
	ErrUnsupported = errors.New("The requested operation is not supported by the flash")
)

type spiflash struct {
	mem memProvider
	tCK int
}

type commonSpiFlash struct {
	*spiflash
}

type mx25l256 struct {
	*commonSpiFlash
}

type mt25q512 struct {
	*commonSpiFlash
}

type Flash interface {
	Id() uint32
	Close()
	Read([]byte) (int, error)
	ReadAt([]byte, int64) (int, error)
	Write([]byte) (int, error)
	WriteAt([]byte, int64) (int, error)
	LockBootloader() error
	StatusFlags() (int, error)
}

func (f *spiflash) cs(h int) {
	// Control CS# (but go doesn't allow # in function names so the name is
	// a bit confusing.

	// Set tCK for clock divider, enable user mode, and set CS# to argument
	cr := uint32(f.tCK&0x0f<<8 | 0x3 | h<<2)
	f.mem.MustWrite32(CS0_CTRL, cr)
}

func (f *spiflash) status() uint8 {
	return f.cmd8Read8(OP_READ_STATUS)
}

func (f *spiflash) isReady() bool {
	return f.status()&0x1 == 0
}

func (f *spiflash) Id() uint32 {
	// Hopefully OP_ID is standard enough that this works fine to identify
	// various SPI flashes that might be out there
	return f.cmd8Read32(OP_ID) & 0xffffff
}

func (f *spiflash) cmd8(cmd int) {
	f.cs(0)
	defer f.cs(1)
	f.mem.MustWrite8(FLASH_START, uint8(cmd&0xff))
}

func (f *spiflash) cmd8Read32(cmd int) uint32 {
	f.cs(0)
	defer f.cs(1)
	f.mem.MustWrite8(FLASH_START, uint8(cmd&0xff))
	return f.mem.MustRead32(FLASH_START)
}

func (f *spiflash) cmd8Read8(cmd int) uint8 {
	f.cs(0)
	defer f.cs(1)
	f.mem.MustWrite8(FLASH_START, uint8(cmd&0xff))
	return f.mem.MustRead8(FLASH_START)
}

func (a *Ast) SystemFlash() (Flash, error) {
	mem := a.Mem()
	// Assume SPI flash
	// Reset CE0
	mem.MustWrite32(CS0_CTRL, 0)
	mem.MustWrite32(CS0_SEGMENT_ADDR, 0x48400000) // See manual for reset value
	mem.MustWrite32(SPI_READ_TIMINGS, 0)

	// Read ID with low clock to maximize the odds of reading the ID correctly
	// for devices we do not know about
	f := spiflash{mem, 0}
	for !f.isReady() {
		time.Sleep(100 * time.Millisecond)
	}
	id := f.Id()
	if id == MX25L256_ID {
		return newMX25L256Flash(a), nil
	} else if id == MT25Q512_ID {
		return newMT25Q512Flash(a), nil
	} else {
		return nil, fmt.Errorf("Unknown flash ID: %06x", id)
	}
}

func (f *commonSpiFlash) StatusFlags() (int, error) {
	f.cs(0)
	defer f.cs(1)
	f.mem.MustWrite8(FLASH_START, uint8(COMMON_OP_RD_FLAG_REG&0xff))
	return int(f.mem.MustRead8(FLASH_START)), nil
}

func (f *commonSpiFlash) Read(b []byte) (int, error) {
	return f.ReadAt(b, 0)
}

func (f *commonSpiFlash) ReadAt(b []byte, off int64) (int, error) {
	l := len(b)
	if off+int64(l) > 32*1024*1024 {
		return 0, fmt.Errorf("Read would have overflown chip")
	}
	f.cs(0)
	defer f.cs(1)
	f.mem.MustWrite8(FLASH_START, uint8(COMMON_OP_FAST_READ&0xff))
	f.mem.MustWrite8(FLASH_START, uint8(off>>24&0xff))
	f.mem.MustWrite8(FLASH_START, uint8(off>>16&0xff))
	f.mem.MustWrite8(FLASH_START, uint8(off>>8&0xff))
	f.mem.MustWrite8(FLASH_START, uint8(off&0xff))
	f.mem.MustWrite8(FLASH_START, 0) // 8 dummy cycles
	i := 0
	for ; i < l-3; i += 4 {
		d := f.mem.MustRead32(FLASH_START)
		b[i] = byte(d & 0xff)
		b[i+1] = byte(d >> 8 & 0xff)
		b[i+2] = byte(d >> 16 & 0xff)
		b[i+3] = byte(d >> 24 & 0xff)
	}
	for i < l {
		d := f.mem.MustRead32(FLASH_START)
		b[i] = byte(d & 0xff)
		i += 1
		if i < l {
			b[i] = byte(d >> 8 & 0xff)
			i += 1
		}
		if i < l {
			b[i] = byte(d >> 16 & 0xff)
			i += 1
		}
		if i < l {
			b[i] = byte(d >> 24 & 0xff)
			i += 1
		}
	}
	return i, nil
}

func (f *commonSpiFlash) Write(b []byte) (int, error) {
	return f.WriteAt(b, 0)
}

func (f *commonSpiFlash) eraseBlock(b int32) {
	f.cmd8(COMMON_OP_WREN)
	f.cs(0)
	f.mem.MustWrite8(FLASH_START, uint8(COMMON_OP_BLOCK_ERASE&0xff))
	f.mem.MustWrite8(FLASH_START, uint8(b>>24&0xff))
	f.mem.MustWrite8(FLASH_START, uint8(b>>16&0xff))
	f.mem.MustWrite8(FLASH_START, uint8(0)) // Blocks are 64kb, lower 16b are 0
	f.mem.MustWrite8(FLASH_START, uint8(0))
	f.cs(1)

	for !f.isReady() {
		time.Sleep(time.Millisecond)
	}
}

func (f *commonSpiFlash) programPage(p int32, d []byte) {
	if len(d) != 256 {
		panic("Expected 256 byte page block")
	}
	f.cmd8(COMMON_OP_WREN)
	f.cs(0)
	f.mem.MustWrite8(FLASH_START, uint8(COMMON_OP_PAGE_PROGRAM&0xff))
	f.mem.MustWrite8(FLASH_START, uint8(p>>24&0xff))
	f.mem.MustWrite8(FLASH_START, uint8(p>>16&0xff))
	f.mem.MustWrite8(FLASH_START, uint8(p>>8&0xff))
	f.mem.MustWrite8(FLASH_START, uint8(0)) // Pages are 256 byte, lower 8b are 0
	for i := 0; i < len(d); i += 4 {
		v := uint32(d[i])
		v |= uint32(d[i+1]) << 8
		v |= uint32(d[i+2]) << 16
		v |= uint32(d[i+3]) << 24
		f.mem.MustWrite32(FLASH_START, v)
	}
	f.cs(1)

	for !f.isReady() {
		time.Sleep(time.Millisecond)
	}
}

func (f *commonSpiFlash) WriteAt(b []byte, off int64) (int, error) {
	// TODO(bluecmd): It's not that hard to support non-64k aligned writes,
	// so we might do that at some point
	l := len(b)
	if l%1024*64 != 0 {
		return 0, fmt.Errorf("Buffer needs to be multiple of 64KB")
	}
	if off < 0 || off%1024*64 != 0 {
		return 0, fmt.Errorf("Offset needs to be positive multiple of 64KB")
	}
	if off+int64(l) > 32*1024*1024 {
		return 0, fmt.Errorf("Write would have overflown chip")
	}

	for i := off; i < off+int64(l); i += 64 * 1024 {
		f.eraseBlock(int32(i))
	}

	for i := off; i < off+int64(l); i += 256 {
		f.programPage(int32(i), b[i-off:i-off+256])
	}

	return l, nil
}

func newMX25L256Flash(a *Ast) *mx25l256 {
	// 6 is /4 which is the fastest that has worked while developing
	// ASPEED's socflash uses /4 (value 6) and /13 (value 0xb)
	// When trying higher clockspeeds the SPI flash got confused and stopped
	// working, so be careful when tuning this.
	f := mx25l256{&commonSpiFlash{&spiflash{a.Mem(), 6}}}
	// Use 4 byte mode
	f.cmd8(COMMON_OP_EN4B)
	return &f
}

func (f *mx25l256) LockBootloader() error {
	return ErrUnsupported
}

func (f *mx25l256) Close() {
	f.cmd8(COMMON_OP_EX4B)
}

func newMT25Q512Flash(a *Ast) *mt25q512 {
	// TODO(bluecmd): Figure out max clock for this chip
	f := mt25q512{&commonSpiFlash{&spiflash{a.Mem(), 6}}}
	// Use 4 byte mode
	f.cmd8(COMMON_OP_EN4B)
	return &f
}

func (f *mt25q512) Close() {
	f.cmd8(COMMON_OP_EX4B)
}

func waitForReady(f Flash) {
	for {
		if v, _ := f.StatusFlags(); v & 0x80 != 0 {
			return
		}
	}
}

func (f *mt25q512) LockBootloader() error {
	lv := uint8(0x3) // Enable write lock and lock down
	// Lock first 512 KiB
	// The first sector has 4K sub-pages
	i := 0
	for ; i < 64 * 1024; i += 4 * 1024 {
		waitForReady(f)
		f.cmd8(COMMON_OP_WREN)
		f.cs(0)
		f.mem.MustWrite8(FLASH_START, uint8(MT25Q_WR_LOCK_BITS&0xff))
		f.mem.MustWrite8(FLASH_START, uint8((i >> 24) & 0xff))
		f.mem.MustWrite8(FLASH_START, uint8((i >> 16) & 0xff))
		f.mem.MustWrite8(FLASH_START, uint8((i >> 8) & 0xff))
		f.mem.MustWrite8(FLASH_START, uint8(i & 0xff))
		f.mem.MustWrite8(FLASH_START, uint8(lv))
		f.cs(1)
	}
	// The next are normal 64K
	for ; i < 512 * 1024; i += 64 * 1024 {
		waitForReady(f)
		f.cmd8(COMMON_OP_WREN)
		f.cs(0)
		// Lock first 512 KiB
		f.mem.MustWrite8(FLASH_START, uint8(MT25Q_WR_LOCK_BITS&0xff))
		f.mem.MustWrite8(FLASH_START, uint8((i >> 24) & 0xff))
		f.mem.MustWrite8(FLASH_START, uint8((i >> 16) & 0xff))
		f.mem.MustWrite8(FLASH_START, uint8((i >> 8) & 0xff))
		f.mem.MustWrite8(FLASH_START, uint8(i & 0xff))
		f.mem.MustWrite8(FLASH_START, uint8(lv))
		f.cs(1)
	}

	// Verify
	waitForReady(f)
	f.cs(0)
	f.mem.MustWrite8(FLASH_START, uint8(MT25Q_RD_LOCK_BITS&0xff))
	f.mem.MustWrite8(FLASH_START, uint8(0))
	f.mem.MustWrite8(FLASH_START, uint8(0))
	f.mem.MustWrite8(FLASH_START, uint8(0))
	f.mem.MustWrite8(FLASH_START, uint8(0))

	ok := true
	for i := 0; i < 512*1024; i++ {
		r := f.mem.MustRead8(FLASH_START)
		if r & 0x3 != lv {
			log.Printf("! %08x: %02x", i, r)
			ok = false
		}
	}
	f.cs(1)

	if !ok {
		return fmt.Errorf("Verification of locking failed")
	}
	return nil
}
