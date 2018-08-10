// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ast2400

import (
	"os"
	"syscall"
	"unsafe"
)

type hostMem struct {
	mf *os.File
}

func openHostMemory() *hostMem {
	f, err := os.OpenFile("/dev/mem", os.O_RDWR|os.O_SYNC, 0600)
	if err != nil {
		panic(err)
	}
	return &hostMem{f}
}

// TODO(bluecmd): Mapping and unmapping like this is not going to fly if
// we need to do any non-trivial data pushing. Leave it for now, but
// will need to re-think later.
func (m *hostMem) MustRead32(address uintptr) uint32 {
	ps := uintptr(syscall.Getpagesize())
	page := (address & ^(ps - 1))
	offset := address - page
	mem, err := syscall.Mmap(int(m.mf.Fd()), int64(page), int(ps), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		panic(err)
	}
	v := *(*uint32)(unsafe.Pointer(&mem[offset]))
	err = syscall.Munmap(mem)
	if err != nil {
		panic(err)
	}
	return v
}

func (m *hostMem) MustRead8(address uintptr) uint8 {
	ps := uintptr(syscall.Getpagesize())
	page := (address & ^(ps - 1))
	offset := address - page
	mem, err := syscall.Mmap(int(m.mf.Fd()), int64(page), int(ps), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		panic(err)
	}
	v := *(*uint8)(unsafe.Pointer(&mem[offset]))
	err = syscall.Munmap(mem)
	if err != nil {
		panic(err)
	}
	return v
}

func (m *hostMem) MustWrite32(address uintptr, data uint32) {
	ps := uintptr(syscall.Getpagesize())
	page := (address & ^(ps - 1))
	offset := address - page
	mem, err := syscall.Mmap(int(m.mf.Fd()), int64(page), int(ps), syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		panic(err)
	}
	*(*uint32)(unsafe.Pointer(&mem[offset])) = data
	err = syscall.Munmap(mem)
	if err != nil {
		panic(err)
	}
}

func (m *hostMem) MustWrite8(address uintptr, data uint8) {
	ps := uintptr(syscall.Getpagesize())
	page := (address & ^(ps - 1))
	offset := address - page
	mem, err := syscall.Mmap(int(m.mf.Fd()), int64(page), int(ps), syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		panic(err)
	}
	*(*uint8)(unsafe.Pointer(&mem[offset])) = data
	err = syscall.Munmap(mem)
	if err != nil {
		panic(err)
	}
}

func (m *hostMem) Close() {
	m.Close()
}
