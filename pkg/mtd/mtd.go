// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mtd

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"syscall"
	"unsafe"
)

const (
	MEMGETINFO = 0x80204d01
	MEMUNLOCK  = 0x40084d06
	MEMERASE   = 0x40084d02
)

type mtdInfoUser struct {
	Type uint8
	// C struct is not packed, so type is padded in 32 bit
	_         uint8
	_         uint8
	_         uint8
	Flags     uint32
	Size      uint32
	EraseSize uint32
	WriteSize uint32
	OobSize   uint32
	// Deprecated fields that are not part of the structure any longer
	_ uint32
	_ uint32
}

type eraseInfoUser struct {
	Start  uint32
	Length uint32
}

type mtdFile struct {
	f         *os.File
	Size      int64
	EraseSize int64
	WriteSize int64
}

func (m *mtdFile) ioctl(req uint, arg []byte) error {
	argp := uintptr(unsafe.Pointer(&arg[0]))
	_, _, e := syscall.Syscall(syscall.SYS_IOCTL, m.f.Fd(), uintptr(req), argp)
	if e != 0 {
		return os.NewSyscallError("ioctl", e)
	}
	return nil
}

func Open(path string) (*mtdFile, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_SYNC, 0600)
	if err != nil {
		return nil, err
	}

	m := &mtdFile{}
	m.f = f

	info := mtdInfoUser{}
	ib := make([]byte, unsafe.Sizeof(info))
	err = m.ioctl(MEMGETINFO, ib)
	if err != nil {
		return m, nil
	}
	buf := bytes.NewReader(ib)
	err = binary.Read(buf, binary.LittleEndian, &info)
	if err != nil {
		return nil, err
	}

	m.Size = int64(info.Size)
	m.EraseSize = int64(info.EraseSize)
	// Write in erase block sizes to be nice to the flash
	m.WriteSize = int64(info.EraseSize)

	return m, nil
}

func (m *mtdFile) Erase() {
	var i uint32
	for i = 0; i < uint32(m.Size); i += uint32(m.EraseSize) {
		ei := eraseInfoUser{i, uint32(m.EraseSize)}
		buf := new(bytes.Buffer)
		err := binary.Write(buf, binary.LittleEndian, ei)
		if err != nil {
			panic(err)
		}
		// TODO(bluecmd) AMI BMC doesn't support UNLOCK even though it's supposed
		// to be mandatory for erase? Oh well.
		//err = m.ioctl(MEMUNLOCK, buf.Bytes())
		//if err != nil {
		//	panic(err)
		//}
		err = m.ioctl(MEMERASE, buf.Bytes())
		if err != nil {
			panic(err)
		}
	}
}

func (m *mtdFile) Write(f io.Reader) {
	m.f.Seek(0, 0)
	buf := make([]byte, m.WriteSize)
	_, err := io.CopyBuffer(m.f, f, buf)
	if err != nil {
		panic(err)
	}
}

func (m *mtdFile) Verify(f io.Reader) bool {
	m.f.Seek(0, 0)
	buf1 := make([]byte, m.WriteSize)
	buf2 := make([]byte, m.WriteSize)
	for {
		n1, err1 := f.Read(buf1)
		n2, err2 := m.f.Read(buf2)

		if n1 != n2 {
			return false
		}

		if !bytes.Equal(buf1, buf2) {
			return false
		}

		if err1 != nil && err2 != io.EOF {
			return false
		}
		if err2 != nil && err2 != io.EOF {
			return false
		}

		if err1 == io.EOF && err2 == io.EOF {
			break
		}
	}
	return true
}
