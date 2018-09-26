// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"os"
	"net"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv6"
)

func ping(host string, iface string) error {
	// From golang.org/x/net/icmp example
	c, err := icmp.ListenPacket("udp6", "::")
	if err != nil {
		return fmt.Errorf("ListenPacket: %v", err)
	}
	defer c.Close()

	wm := icmp.Message{
		Type: ipv6.ICMPTypeEchoRequest, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
			Data: []byte("icmp-test"),
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		return fmt.Errorf("wm.Marshal: %v", err)
	}
	if _, err := c.WriteTo(wb, &net.UDPAddr{IP: net.ParseIP(host), Zone: iface}); err != nil {
		return fmt.Errorf("WriteTo: %v", err)
	}

	rb := make([]byte, 1500)
	n, peer, err := c.ReadFrom(rb)
	if err != nil {
		return fmt.Errorf("ReadFrom: %v", err)
	}
	rm, err := icmp.ParseMessage(58, rb[:n])
	if err != nil {
		return fmt.Errorf("icmp.ParseMessage: %v", err)
	}
	switch rm.Type {
	case ipv6.ICMPTypeEchoReply:
		log.Printf("got reflection from %v", peer)
		return nil
	default:
		return fmt.Errorf("got %+v; want echo reply", rm)
	}
}
