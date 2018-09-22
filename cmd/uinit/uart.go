// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"

	"github.com/tarm/serial"
)

var (
	// Read from the host
	uartIn  chan []byte
	// To be written to the host
	uartOut chan []byte
)

func uartSender(s *serial.Port) {
	for {
		buf := <-uartOut
		_, err := s.Write(buf)
		if err != nil {
			log.Printf("UART write error: %v", err)
			break
		}
	}
}
func startUart(f string) {
	// TODO(bluecmd): This is platform specific
	c := &serial.Config{Name: f, Baud: 57600}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Printf("serial.OpenPort: %v", err)
		return
	}

	go uartSender(s)

	buf := make([]byte, 128)
	for {
		n, err := s.Read(buf)
		if err != nil {
			log.Printf("UART read error: %v", err)
			break
		}
		select {
		case uartIn <- buf[:n]:
		default:
			// TODO(bluecmd): This would be good to buffer
		}
	}
}
