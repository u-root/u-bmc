// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ast2400

import (
	"bytes"
	"testing"
)

func expectInit(f *fakeMem, chip uint32) {
	f.ExpectWrite32(0x1e620010, 0)
	f.ExpectWrite32(0x1e620030, 0x48400000)
	f.ExpectWrite32(0x1e620094, 0)

	// Write-in-progress
	f.ExpectWrite32(0x1e620010, 0x3)
	f.ExpectWrite8(0x20000000, 0x05)
	f.FakeRead8(0x20000000, 1)
	f.ExpectWrite32(0x1e620010, 0x7)

	// Ready
	f.ExpectWrite32(0x1e620010, 0x3)
	f.ExpectWrite8(0x20000000, 0x05)
	f.FakeRead8(0x20000000, 0)
	f.ExpectWrite32(0x1e620010, 0x7)

	// ID read
	f.ExpectWrite32(0x1e620010, 0x3)
	f.ExpectWrite8(0x20000000, 0x9f)
	f.FakeRead32(0x20000000, 0xff000000|chip)
	f.ExpectWrite32(0x1e620010, 0x7)
}

func expectCmd8(f *fakeMem, cmd uint8) {
	f.ExpectWrite32(0x1e620010, 0x603)
	f.ExpectWrite8(0x20000000, cmd)
	f.ExpectWrite32(0x1e620010, 0x607)
}

func expectCmd8Read32(f *fakeMem, cmd uint8, resp uint32) {
	f.ExpectWrite32(0x1e620010, 0x603)
	f.ExpectWrite8(0x20000000, cmd)
	f.FakeRead32(0x20000000, resp)
	f.ExpectWrite32(0x1e620010, 0x607)
}

func TestMx25l256Supported(t *testing.T) {
	fm := fakeMemory(t)
	a := OpenWithMemory(fm)
	expectInit(fm, MX25L256_ID)
	expectCmd8(fm, COMMON_OP_EN4B)
	f, err := a.SystemFlash()
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}

	expectCmd8Read32(fm, 0x9f, MX25L256_ID)
	id := f.Id()
	if id != MX25L256_ID {
		t.Fatalf("ID verification failed %v != %v", id, MX25L256_ID)
	}

	expectCmd8(fm, COMMON_OP_EX4B)
	f.Close()
}

func TestUnknwonNotSupported(t *testing.T) {
	fm := fakeMemory(t)
	a := OpenWithMemory(fm)
	expectInit(fm, 0xdeadbe)
	_, err := a.SystemFlash()
	if err == nil {
		t.Fatalf("Unknown flash reported as supported")
	}
}

func TestMx25l256FastRead(t *testing.T) {
	fm := fakeMemory(t)
	a := OpenWithMemory(fm)
	expectInit(fm, MX25L256_ID)
	expectCmd8(fm, COMMON_OP_EN4B)
	f, err := a.SystemFlash()
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}

	fm.ExpectWrite32(0x1e620010, 0x603)
	fm.ExpectWrite8(0x20000000, COMMON_OP_FAST_READ)
	fm.ExpectWrite8(0x20000000, 0x01)
	fm.ExpectWrite8(0x20000000, 0x55)
	fm.ExpectWrite8(0x20000000, 0xbb)
	fm.ExpectWrite8(0x20000000, 0xcc)
	fm.ExpectWrite8(0x20000000, 0)
	fm.FakeRead32(0x20000000, 0x04030201)
	fm.FakeRead32(0x20000000, 0x08070605)
	fm.ExpectWrite32(0x1e620010, 0x607)

	b := make([]byte, 7)
	n, err := f.ReadAt(b, 0x0155bbcc)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != 7 {
		t.Fatalf("Expected 7 bytes read, got %v\n", n)
	}

	e := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}
	if !bytes.Equal(e, b) {
		t.Fatalf("Expected %v, got %v", e, b)
	}
}

func TestMx25l256EraseAndWrite(t *testing.T) {
	fm := fakeMemory(t)
	a := OpenWithMemory(fm)
	expectInit(fm, MX25L256_ID)
	expectCmd8(fm, COMMON_OP_EN4B)
	f, err := a.SystemFlash()
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}

	expectCmd8(fm, COMMON_OP_WREN)
	fm.ExpectWrite32(0x1e620010, 0x603)
	fm.ExpectWrite8(0x20000000, COMMON_OP_BLOCK_ERASE)
	fm.ExpectWrite8(0x20000000, 0x01)
	fm.ExpectWrite8(0x20000000, 0x55)
	fm.ExpectWrite8(0x20000000, 0x00)
	fm.ExpectWrite8(0x20000000, 0x00)
	fm.ExpectWrite32(0x1e620010, 0x607)

	// Write-in-progress
	fm.ExpectWrite32(0x1e620010, 0x603)
	fm.ExpectWrite8(0x20000000, 0x05)
	fm.FakeRead8(0x20000000, 1)
	fm.ExpectWrite32(0x1e620010, 0x607)

	// Ready
	fm.ExpectWrite32(0x1e620010, 0x603)
	fm.ExpectWrite8(0x20000000, 0x05)
	fm.FakeRead8(0x20000000, 0)
	fm.ExpectWrite32(0x1e620010, 0x607)

	for i := 0; i < 64*1024/256; i++ {
		expectCmd8(fm, COMMON_OP_WREN)
		fm.ExpectWrite32(0x1e620010, 0x603)
		fm.ExpectWrite8(0x20000000, COMMON_OP_PAGE_PROGRAM)
		fm.ExpectWrite8(0x20000000, 0x01)
		fm.ExpectWrite8(0x20000000, 0x55)
		fm.ExpectWrite8(0x20000000, uint8(i))
		fm.ExpectWrite8(0x20000000, 0x00)
		// For every page, expect a page program
		for j := 0; j < 256/4; j++ {
			fm.ExpectWrite32(0x20000000, 0)
		}
		fm.ExpectWrite32(0x1e620010, 0x607)

		// Write-in-progress
		fm.ExpectWrite32(0x1e620010, 0x603)
		fm.ExpectWrite8(0x20000000, 0x05)
		fm.FakeRead8(0x20000000, 1)
		fm.ExpectWrite32(0x1e620010, 0x607)

		// Ready
		fm.ExpectWrite32(0x1e620010, 0x603)
		fm.ExpectWrite8(0x20000000, 0x05)
		fm.FakeRead8(0x20000000, 0)
		fm.ExpectWrite32(0x1e620010, 0x607)
	}

	b := make([]byte, 64*1024)
	n, err := f.WriteAt(b, 0x01550000)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != 64*1024 {
		t.Fatalf("Expected 64 kbytes read, got %v\n", n)
	}
}
