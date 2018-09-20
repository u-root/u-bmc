// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/binary"
	"io"
	"log"
	"os"

	"github.com/u-root/u-bmc/pkg/ast2400"
)

type playback struct {
	f *os.File
}

func (p *playback) Close() {
	p.f.Close()
}

func (p *playback) SnapshotGpio() *ast2400.State {
	var gpios uint32
	var scus uint32
	var unix int64
	// TODO(bluecmd): Use the unix timestamp to print the sample time
	err := binary.Read(p.f, binary.LittleEndian, &unix)
	if err == io.EOF {
		return nil
	}
	if err != nil {
		log.Fatalf("binary.Read failed:", err)
	}
	err = binary.Read(p.f, binary.LittleEndian, &gpios)
	if err != nil {
		log.Fatalf("binary.Read failed:", err)
	}
	err = binary.Read(p.f, binary.LittleEndian, &scus)
	if err != nil {
		log.Fatalf("binary.Read failed:", err)
	}

	s := ast2400.State{make(map[uint32]uint32), make(map[uint32]uint32)}
	for i := uint32(0); i < gpios; i++ {
		var k uint32
		var v uint32
		err = binary.Read(p.f, binary.LittleEndian, &k)
		if err != nil {
			log.Fatalf("binary.Read failed:", err)
		}
		err = binary.Read(p.f, binary.LittleEndian, &v)
		if err != nil {
			log.Fatalf("binary.Read failed:", err)
		}
		s.Gpio[k] = v
	}
	for i := uint32(0); i < scus; i++ {
		var k uint32
		var v uint32
		err = binary.Read(p.f, binary.LittleEndian, &k)
		if err != nil {
			log.Fatalf("binary.Read failed:", err)
		}
		err = binary.Read(p.f, binary.LittleEndian, &v)
		if err != nil {
			log.Fatalf("binary.Read failed:", err)
		}
		s.Scu[k] = v
	}
	return &s
}
