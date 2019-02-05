// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpiowatcher

import (
	"encoding/binary"
	"log"
	"os"
	"time"

	"github.com/u-root/u-bmc/pkg/aspeed"
)

type binaryLog struct {
	f *os.File
}

func newBinaryLog(p *aspeed.State) *binaryLog {
	l := binaryLog{os.Stdout}
	l.Log(p)
	return &l
}

func (l *binaryLog) Log(s *aspeed.State) {
	err := binary.Write(l.f, binary.LittleEndian, int64(time.Now().Unix()))
	if err != nil {
		log.Fatalf("binary.Write failed: %v", err)
	}
	err = binary.Write(l.f, binary.LittleEndian, uint32(len(s.Gpio)))
	if err != nil {
		log.Fatalf("binary.Write failed: %v", err)
	}
	err = binary.Write(l.f, binary.LittleEndian, uint32(len(s.Scu)))
	if err != nil {
		log.Fatalf("binary.Write failed: %v", err)
	}
	for k, v := range s.Gpio {
		err = binary.Write(l.f, binary.LittleEndian, k)
		if err != nil {
			log.Fatalf("binary.Write failed: %v", err)
		}
		err = binary.Write(l.f, binary.LittleEndian, v)
		if err != nil {
			log.Fatalf("binary.Write failed: %v", err)
		}
	}
	for k, v := range s.Scu {
		err = binary.Write(l.f, binary.LittleEndian, k)
		if err != nil {
			log.Fatalf("binary.Write failed: %v", err)
		}
		err = binary.Write(l.f, binary.LittleEndian, v)
		if err != nil {
			log.Fatalf("binary.Write failed: %v", err)
		}
	}

}

func (l *binaryLog) Close() {
	l.f.Close()
}
