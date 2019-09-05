// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/vishvananda/netlink/nl"
	"golang.org/x/sys/unix"
)

const (
	sizeofNDUseroptmsg = 16
	sizeofNDUseropthdr = 4

	ND_ROUTER_ADVERT = 134
	ND_OPT_RDNSS     = 25
)

// TODO(bluecmd): This should probably be moved to x/sys/unix
type ndUseroptmsg struct {
	Family   uint8
	_        uint8
	OptsLen  uint16
	Ifindex  uint32
	IcmpType uint8
	IcmpCode uint8
	_        uint16
	_        uint32
}

type ndUseropthdr struct {
	Type   uint8
	Length uint8
	_      uint16
}

type NDUserOpt struct {
	Family   uint8
	Ifindex  uint32
	ICMPType uint8
	ICMPCode uint8
	RDNSS    []*RDNSSOption
}

type RDNSSOption struct {
	Lifetime time.Duration
	Server   []*net.IP
}

func rdnss(c chan<- *RDNSSOption) {
	s, err := nl.Subscribe(unix.NETLINK_ROUTE, unix.RTNLGRP_ND_USEROPT)
	if err != nil {
		log.Printf("failed to subscribe to RDNSS updates: %v", err)
		return
	}
	defer s.Close()
	for {
		msgs, _, err := s.Receive()
		if err != nil {
			log.Printf("netlink error on rdnss loop: %v", err)
			return
		}
		for _, m := range msgs {
			if m.Header.Type != unix.RTM_NEWNDUSEROPT {
				continue
			}

			opt, err := parseNDUserOpt(m.Data)
			if err != nil {
				log.Printf("error processing nd user opt: %v", err)
			}

			if opt == nil {
				// The packet was successfully parsed but should be ignored
				continue
			}
			if opt.Family != unix.AF_INET6 {
				continue
			}
			if opt.ICMPCode != 0 {
				continue
			}
			if opt.ICMPType != ND_ROUTER_ADVERT {
				continue
			}
			for _, r := range opt.RDNSS {
				c <- r
			}
		}
	}
}

func parseNDUserOpt(b []byte) (*NDUserOpt, error) {
	if len(b) < sizeofNDUseroptmsg {
		return nil, fmt.Errorf("message too short %d < %d", len(b), sizeofNDUseroptmsg)
	}
	msg := ndUseroptmsg{}
	err := binary.Read(bytes.NewBuffer(b[:sizeofNDUseroptmsg]), NativeEndian(), &msg)
	b = b[sizeofNDUseroptmsg:]
	if err != nil {
		return nil, err
	}
	r := &NDUserOpt{
		ICMPCode: msg.IcmpCode,
		ICMPType: msg.IcmpType,
		Ifindex:  msg.Ifindex,
		Family:   msg.Family,
	}

	if len(b) < int(msg.OptsLen) {
		return nil, fmt.Errorf("message too short for options, %d < %d", len(b), msg.OptsLen)
	}
	b = b[:msg.OptsLen]

	// TODO(bluecmd): ndisc6 loops over the message received from the kernel,
	// and looking at what the kernel gives back it actually gives us one
	// 25 (RDNSS) and one 20 (Neighbor Advertisement Acknowledgment) with size 0.
	// Size 0 should never happen, and looking at the OptsLen we shouldn't read it.
	// I'll do the naive thing and loop over what the kernel tells us to loop
	// over, if this eve breaks we need to look at it closer.
	for {
		msg := ndUseropthdr{}
		err := binary.Read(bytes.NewBuffer(b[:sizeofNDUseropthdr]), binary.BigEndian, &msg)
		if err != nil {
			return nil, err
		}
		l := int(msg.Length) << 3
		if l == 0 {
			// RFC 4861 specifies that a packet with length 0 is invalid, so ignore it
			return nil, nil
		}
		if len(b) < l {
			return nil, fmt.Errorf("message ran out while reading options")
		}
		if msg.Type == ND_OPT_RDNSS {
			rs, err := parseRDNSSOption(b[:l])
			if err != nil {
				return nil, err
			}
			r.RDNSS = append(r.RDNSS, rs)
		}

		b = b[l:]
		if len(b) < sizeofNDUseropthdr {
			// No more headers can be read, assume done
			break
		}
	}
	return r, nil
}

func parseRDNSSOption(b []byte) (*RDNSSOption, error) {
	r := &RDNSSOption{}
	if len(b) < 24 {
		return nil, fmt.Errorf("RDNSS option too short %d < 8", len(b))
	}
	// Skip the header
	b = b[4:]
	r.Lifetime = time.Duration(binary.BigEndian.Uint32(b)) * time.Second
	b = b[4:]
	for len(b) >= 16 {
		ip := (net.IP)(b[:16])
		b = b[16:]
		r.Server = append(r.Server, &ip)
	}
	return r, nil
}
