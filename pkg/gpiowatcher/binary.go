// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpiowatcher

import (
	"encoding/binary"
	"log"
	"os"
	"time"

	"github.com/u-root/u-bmc/pkg/ast2400"
)

type binaryLog struct {
	f *os.File
}

func newBinaryLog(p *ast2400.State) *binaryLog {
	l := binaryLog{os.Stdout}
	l.Log(p)
	return &l
}

func (l *binaryLog) Log(s *ast2400.State) {
	err := binary.Write(l.f, binary.LittleEndian, int64(time.Now().Unix()))
	if err != nil {
		log.Fatalf("binary.Write failed:", err)
	}
	err = binary.Write(l.f, binary.LittleEndian, uint32(len(s.Gpio)))
	if err != nil {
		log.Fatalf("binary.Write failed:", err)
	}
	err = binary.Write(l.f, binary.LittleEndian, uint32(len(s.Scu)))
	if err != nil {
		log.Fatalf("binary.Write failed:", err)
	}
	for k, v := range s.Gpio {
		err = binary.Write(l.f, binary.LittleEndian, k)
		if err != nil {
			log.Fatalf("binary.Write failed:", err)
		}
		err = binary.Write(l.f, binary.LittleEndian, v)
		if err != nil {
			log.Fatalf("binary.Write failed:", err)
		}
	}
	for k, v := range s.Scu {
		err = binary.Write(l.f, binary.LittleEndian, k)
		if err != nil {
			log.Fatalf("binary.Write failed:", err)
		}
		err = binary.Write(l.f, binary.LittleEndian, v)
		if err != nil {
			log.Fatalf("binary.Write failed:", err)
		}
	}

}

func (l *binaryLog) Close() {
	l.f.Close()
}
