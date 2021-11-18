// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"fmt"
	"sync"

	"github.com/tarm/serial"
	"github.com/u-root/u-bmc/pkg/metric"
)

type Uart interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
}

type uartSystem struct {
	u       Uart
	m       *sync.Mutex
	readers []*readerStream
	w       chan []byte
}

type readerStream struct {
	done   <-chan struct{}
	stream chan<- []byte
}

type writerStream struct {
	stream <-chan []byte
}

var (
	// Number of UART reads that were dropped due to a slow client
	uartOverruns = metric.Counter(metric.MetricOpts{
		Namespace: "ubmc",
		Subsystem: "uart",
		Name:      "overrun_count",
	}, nil)
	// Approximate number of UART reads that have not been read yet by a consumer
	//TODO(MDr164): Set func correctly
	uartBufferedReads = metric.Gauge(metric.MetricOpts{
		Namespace: "ubmc",
		Subsystem: "uart",
		Name:      "buffered_read_count",
	}, nil, func() float64 { return 0 })
	// How many UART consumers the system has
	//TODO(MDr164): Set func correctly
	uartConsumers = metric.Gauge(metric.MetricOpts{
		Namespace: "ubmc",
		Subsystem: "uart",
		Name:      "consumer_count",
	}, nil, func() float64 { return 0 })
)

func (u *uartSystem) NewReader(done <-chan struct{}) <-chan []byte {
	// TODO(bluecmd): Is buffering 128*1024 = 128 KiB data enough? Probably, some metrics
	// to debug it would be nice. For now we will output console messages
	// if this queue becomes full.
	c := make(chan []byte, 1024)
	//uartConsumers.Inc()
	u.m.Lock()
	defer u.m.Unlock()
	reader := &readerStream{done, c}
	u.readers = append(u.readers, reader)

	go func(reader *readerStream) {
		<-reader.done
		//uartConsumers.Dec()
		u.m.Lock()
		defer u.m.Unlock()
		var nr []*readerStream
		for _, r := range u.readers {
			if r == reader {
				continue
			}
			nr = append(nr, r)
		}
		u.readers = nr
	}(reader)

	return c
}

func (u *uartSystem) NewWriter() chan<- []byte {
	c := make(chan []byte)
	go func() {
		// Simply copy any data written
		for d := range c {
			u.w <- d
		}
	}()
	return c
}

func (u *uartSystem) uartSender() {
	for {
		buf := <-u.w
		_, err := u.u.Write(buf)
		if err != nil {
			log.Errorf("UART write error: %v", err)
			break
		}
	}
}

func (u *uartSystem) uartReceiver() {
	for {
		buf := make([]byte, 128)
		n, err := u.u.Read(buf)
		if err != nil {
			log.Errorf("UART read error: %v", err)
			break
		}

		u.m.Lock()
		rs := u.readers
		u.m.Unlock()
		p := 0
		for _, r := range rs {
			select {
			case r.stream <- buf[:n]:
				continue
			default:
				uartOverruns.Inc()
			}
			p += len(r.stream)
		}
		//uartBufferedReads.Set(float64(p))
		// TODO(bluecmd): Consider saving a certain set of scrollback and
		// implementing some form of sequence numbering to make it possible to
		// save/restore serial data and request missed frames during network flaps
	}
}

func startUart(f string, baud int) (*uartSystem, error) {
	if f == "" {
		return nil, nil
	}
	c := &serial.Config{Name: f, Baud: baud}
	s, err := serial.OpenPort(c)
	if err != nil {
		return nil, fmt.Errorf("serial.OpenPort: %v", err)
	}
	return newUartSystem(s), nil
}

func newUartSystem(u Uart) *uartSystem {
	s := uartSystem{
		u: u,
		m: &sync.Mutex{},
		w: make(chan []byte),
	}

	go s.uartSender()
	go s.uartReceiver()
	return &s
}
