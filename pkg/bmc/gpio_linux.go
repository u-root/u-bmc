// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"encoding/binary"
	"fmt"
	"io"
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

	GPIOEVENT_EVENT_RISING_EDGE  = 1
	GPIOEVENT_EVENT_FALLING_EDGE = 2
)

type gpioLnx struct {
	f *os.File
}

type gpioLnxLine struct {
	f *os.File
}

func (g *gpioLnx) requestLineHandle(lines []uint32, out []bool) (gpioLineImpl, error) {
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
	copy(rinfo.consumer_label[:], []byte("u-bmc"))
	if len(out) > 0 {
		rinfo.flags = GPIOHANDLE_REQUEST_OUTPUT
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(g.f.Fd()),
		uintptr(GPIO_GET_LINEHANDLE_IOCTL),
		uintptr(unsafe.Pointer(&rinfo)))
	if errno != 0 {
		return nil, fmt.Errorf("GPIO_GET_LINEHANDLE_IOCTL: errno %v", errno)
	}
	return &gpioLnxLine{os.NewFile(uintptr(rinfo.fd), "gpio")}, nil
}

func getLineValues(f *os.File) ([]bool, error) {
	hinfo := gpiohandle_data{}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(f.Fd()),
		uintptr(GPIOHANDLE_GET_LINE_VALUES_IOCTL),
		uintptr(unsafe.Pointer(&hinfo)))
	if errno != 0 {
		return nil, fmt.Errorf("GPIOHANDLE_GET_LINE_VALUES_IOCTL: errno %v", errno)
	}

	b := make([]bool, len(hinfo.values))
	for i, v := range hinfo.values {
		if v != 0 {
			b[i] = true
		}
	}
	return b, nil
}

func setLineValues(f *os.File, out []bool) error {
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
		return fmt.Errorf("GPIOHANDLE_SET_LINE_VALUES_IOCTL: errno %v", errno)
	}
	return nil
}

func (l *gpioLnxLine) setValues(out []bool) error {
	return setLineValues(l.f, out)
}

type gpioLnxEvent struct {
	f *os.File
}

func (g *gpioLnx) getLineEvent(line uint32) (gpioEventImpl, error) {
	req := gpioevent_request{}
	req.lineoffset = line
	req.handleflags = uint32(GPIOHANDLE_REQUEST_INPUT)
	req.eventflags = uint32(GPIOEVENT_REQUEST_BOTH_EDGES)
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(g.f.Fd()),
		uintptr(GPIO_GET_LINEEVENT_IOCTL),
		uintptr(unsafe.Pointer(&req)))
	if errno != 0 {
		return nil, fmt.Errorf("GPIO_GET_LINEEVENT_IOCTL: errno %v", errno)
	}
	return &gpioLnxEvent{os.NewFile(uintptr(req.fd), "gpio-line-event")}, nil
}

func (l *gpioLnxEvent) getValue() (bool, error) {
	b, err := getLineValues(l.f)
	if err != nil {
		return false, err
	}
	return b[0], nil
}

func (l *gpioLnxEvent) read() (*int, error) {
	e := gpioevent_data{}
	err := binary.Read(l.f, NativeEndian(), &e)
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("readEvent: %v", err)
	}

	v := GPIO_EVENT_UNKNOWN
	switch e.Id {
	case GPIOEVENT_EVENT_FALLING_EDGE:
		v = GPIO_EVENT_FALLING_EDGE
	case GPIOEVENT_EVENT_RISING_EDGE:
		v = GPIO_EVENT_RISING_EDGE
	default:
		return &v, fmt.Errorf("unknown event: %v", e)
	}
	return &v, nil
}
