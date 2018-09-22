// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"encoding/binary"
	"io"
	"log"
	"os"
	"syscall"
	"unsafe"
)

type gpiochip_info struct {
	name  [32]byte
	label [32]byte
	lines uint32
}

type gpioline_info struct {
	line_offset uint32
	flags       uint32
	name        [32]byte
	consumer    [32]byte
}

type gpiohandle_request struct {
	lineoffsets    [64]uint32
	flags          uint32
	default_values [64]uint8
	consumer_label [32]byte
	lines          uint32
	fd             uint32
}

type gpiohandle_data struct {
	values [64]uint8
}

type gpioevent_request struct {
	lineoffset     uint32
	handleflags    uint32
	eventflags     uint32
	consumer_label [32]byte
	fd             uint32
}

type gpioevent_data struct {
	Timestamp uint64
	Id        uint32
	// Linux wants this structure to be aligned with 16 bytes
	_ uint32
}

const (
	GPIO_GET_CHIPINFO_IOCTL          = 0x8044b401
	GPIO_GET_LINEINFO_IOCTL          = 0xc048b402
	GPIO_GET_LINEHANDLE_IOCTL        = 0xc16cb403
	GPIO_GET_LINEEVENT_IOCTL         = 0xc030b404
	GPIOHANDLE_SET_LINE_VALUES_IOCTL = 0xc040b409
	GPIOHANDLE_GET_LINE_VALUES_IOCTL = 0xc040b408

	GPIOHANDLE_REQUEST_INPUT       = (1 << 0)
	GPIOHANDLE_REQUEST_OUTPUT      = (1 << 1)
	GPIOHANDLE_REQUEST_ACTIVE_LOW  = (1 << 2)
	GPIOHANDLE_REQUEST_OPEN_DRAIN  = (1 << 3)
	GPIOHANDLE_REQUEST_OPEN_SOURCE = (1 << 4)

	GPIOEVENT_REQUEST_RISING_EDGE  = (1 << 0)
	GPIOEVENT_REQUEST_FALLING_EDGE = (1 << 1)
	GPIOEVENT_REQUEST_BOTH_EDGES   = GPIOEVENT_REQUEST_RISING_EDGE | GPIOEVENT_REQUEST_FALLING_EDGE

	GPIOEVENT_EVENT_RISING_EDGE    = 1
	GPIOEVENT_EVENT_FALLING_EDGE   = 2
)

func getLine(f *os.File, line uint32) *gpioline_info {
	linfo := gpioline_info{}
	linfo.line_offset = line
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(f.Fd()),
		uintptr(GPIO_GET_LINEINFO_IOCTL),
		uintptr(unsafe.Pointer(&linfo)))
	if errno != 0 {
		log.Fatalf("GPIO_GET_LINEINFO_IOCTL: errno %v", errno)
	}
	return &linfo
}

func getChip(f *os.File) *gpiochip_info {
	cinfo := gpiochip_info{}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(f.Fd()),
		uintptr(GPIO_GET_CHIPINFO_IOCTL),
		uintptr(unsafe.Pointer(&cinfo)))
	if errno != 0 {
		log.Fatalf("GPIO_GET_CHIPINFO_IOCTL: errno %v", errno)
	}
	return &cinfo
}

func requestLineHandle(f *os.File, lines []uint32, out []bool) *os.File {
	rinfo := gpiohandle_request{}
	for i, l := range lines {
		rinfo.lineoffsets[i] = l
	}
	rinfo.lines = uint32(len(lines))
	for i, l := range out {
		if l {
			rinfo.default_values[i] = 1
		}
	}
	rinfo.flags = GPIOHANDLE_REQUEST_INPUT
	copy(rinfo.consumer_label[:], []byte("gpio-utility"))
	if len(out) > 0 {
		rinfo.flags = GPIOHANDLE_REQUEST_OUTPUT
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(f.Fd()),
		uintptr(GPIO_GET_LINEHANDLE_IOCTL),
		uintptr(unsafe.Pointer(&rinfo)))
	if errno != 0 {
		log.Fatalf("GPIO_GET_LINEHANDLE_IOCTL: errno %v", errno)
	}
	return os.NewFile(uintptr(rinfo.fd), "gpio")
}

func getLineValues(f *os.File) *gpiohandle_data {
	hinfo := gpiohandle_data{}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(f.Fd()),
		uintptr(GPIOHANDLE_GET_LINE_VALUES_IOCTL),
		uintptr(unsafe.Pointer(&hinfo)))
	if errno != 0 {
		log.Fatalf("GPIOHANDLE_GET_LINE_VALUES_IOCTL: errno %v", errno)
	}
	return &hinfo
}

func setLineValues(f *os.File, out []bool) *gpiohandle_data {
	hinfo := gpiohandle_data{}
	for i, v := range out {
		if v {
			hinfo.values[i] = 1
		}
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(f.Fd()),
		uintptr(GPIOHANDLE_SET_LINE_VALUES_IOCTL),
		uintptr(unsafe.Pointer(&hinfo)))
	if errno != 0 {
		log.Fatalf("GPIOHANDLE_SET_LINE_VALUES_IOCTL: errno %v", errno)
	}
	return &hinfo
}

func getLineEvent(f *os.File, line uint32, hf int, ef int) *os.File {
	req := gpioevent_request{}
	req.lineoffset = line
	req.handleflags = uint32(hf)
	req.eventflags = uint32(ef)
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(f.Fd()),
		uintptr(GPIO_GET_LINEEVENT_IOCTL),
		uintptr(unsafe.Pointer(&req)))
	if errno != 0 {
		log.Fatalf("GPIO_GET_LINEEVENT_IOCTL: errno %v", errno)
	}
	return os.NewFile(uintptr(req.fd), "gpio-line-event")
}

func readEvent(f *os.File) *gpioevent_data {
	e := gpioevent_data{}
	err := binary.Read(f, binary.LittleEndian, &e)
	if err == io.EOF {
		return nil
	}
	if err != nil {
		log.Printf("readEvent read: %v", err)
		return nil
	}
	return &e
}
