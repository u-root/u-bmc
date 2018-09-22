// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"fmt"
	"log"

	"github.com/tarm/serial"
)

type uartSystem struct {
	s *serial.Port
	// Read from the host
	uartIn chan []byte
	// To be written to the host
	uartOut chan []byte
}

func (u *uartSystem) Read() []byte {
	return <-u.uartIn
}

func (u *uartSystem) Write(b []byte) {
	u.uartOut <- b
}

func (u *uartSystem) uartSender() {
	for {
		buf := <-u.uartOut
		_, err := u.s.Write(buf)
		if err != nil {
			log.Printf("UART write error: %v", err)
			break
		}
	}
}

func (u *uartSystem) uartReceiver() {
	buf := make([]byte, 128)
	for {
		n, err := u.s.Read(buf)
		if err != nil {
			log.Printf("UART read error: %v", err)
			break
		}
		select {
		case u.uartIn <- buf[:n]:
		default:
			// TODO(bluecmd): This would be good to buffer
		}
	}
}

func startUart(f string, baud int) (*uartSystem, error) {
	c := &serial.Config{Name: f, Baud: baud}
	s, err := serial.OpenPort(c)
	if err != nil {
		return nil, fmt.Errorf("serial.OpenPort: %v", err)
	}

	u := uartSystem{s, make(chan []byte), make(chan []byte)}

	go u.uartSender()
	go u.uartReceiver()
	return &u, nil
}
