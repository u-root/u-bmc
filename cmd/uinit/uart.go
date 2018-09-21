// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"

	"github.com/tarm/serial"
)

func startUart(f string) {
	// TODO(bluecmd): This is platform specific
	c := &serial.Config{Name: f, Baud: 57600}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatalf("serial.OpenPort: %v", err)
	}

	buf := make([]byte, 128)
	for {
		n, err := s.Read(buf)
		if err != nil {
			log.Printf("UART read error: %v", err)
			break
		}
		log.Printf("UART %s: %q", f, buf[:n])
	}
}
