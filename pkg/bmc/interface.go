// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

const (
	interfaceUpTimeout = 30 * time.Second
)

func addIp(cidr string, iface string) error {
	l, err := netlink.LinkByName(iface)
	if err != nil {
		return fmt.Errorf("Unable to get interface %s: %v", iface, err)
	}
	addr, err := netlink.ParseAddr(cidr)
	if err != nil {
		return fmt.Errorf("netlink.ParseAddr %v: %v", cidr, err)
	}
	h, err := netlink.NewHandle(unix.NETLINK_ROUTE)
	if err != nil {
		return fmt.Errorf("netlink.NewHandle: %v", err)
	}
	defer h.Delete()
	if err := h.AddrReplace(l, addr); err != nil {
		return fmt.Errorf("AddrReplace(%v): %v", addr, err)
	}
	return nil
}

func setLinkUp(iface string) error {
	l, err := netlink.LinkByName(iface)
	if err != nil {
		return fmt.Errorf("Unable to get interface %s: %v", iface, err)
	}
	h, err := netlink.NewHandle(unix.NETLINK_ROUTE)
	if err != nil {
		return fmt.Errorf("netlink.NewHandle: %v", err)
	}
	defer h.Delete()
	if err := h.LinkSetUp(l); err != nil {
		return fmt.Errorf("handle.LinkSetUp: %v", err)
	}
	return nil
}

func setLinkDown(iface string) error {
	l, err := netlink.LinkByName(iface)
	if err != nil {
		return fmt.Errorf("Unable to get interface %s: %v", iface, err)
	}
	h, err := netlink.NewHandle(unix.NETLINK_ROUTE)
	if err != nil {
		return fmt.Errorf("netlink.NewHandle: %v", err)
	}
	defer h.Delete()
	if err := h.LinkSetDown(l); err != nil {
		return fmt.Errorf("handle.LinkSetDown: %v", err)
	}
	return nil
}

func isLinkLocalForMAC(addr []byte, hw []byte) bool {
	chw := append(
		[]byte{0xfe, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		hw[0], hw[1], hw[2], 0xff, 0xfe, hw[3], hw[4], hw[5])
	chw[8] ^= 0x2
	return bytes.Equal(chw, addr)
}

func ipv6LinkFixer(iface string) {
	// Verify that the link local address based on the MAC address is present
	// every 1 second. If it's not, reset the interface (and in the future
	// of course reset DHCP etc.).
	h, err := netlink.NewHandle(unix.NETLINK_ROUTE)
	if err != nil {
		log.Printf("netlink.NewHandle: %v", err)
		return
	}
	defer h.Delete()

	sleep := 1 * time.Second
	for {
		time.Sleep(sleep)
		sleep = 1 * time.Second
		l, err := netlink.LinkByName(iface)
		if err != nil {
			log.Printf("Unable to get interface %s: %v", iface, err)
			continue
		}
		a := l.Attrs()
		// OperState does not work with all interfaces, so use flags
		// NC-SI reports 'unknown' for all states (and so does loopback interfaces, FWIW)
		if a.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := h.AddrList(l, netlink.FAMILY_V6)
		if err != nil {
			log.Printf("handle.AddrList(%s): %v", iface, err)
			continue
		}
		found := false
		for _, addr := range addrs {
			if addr.IP.IsLinkLocalUnicast() && isLinkLocalForMAC(addr.IP.To16(), a.HardwareAddr) {
				found = true
				break
			}
		}
		if !found {
			log.Printf("No link-local IPv6 address found for %s, resetting interface", iface)
			setLinkDown(iface)
			setLinkUp(iface)
			// Back off 10 seconds before trying again to avoid flapping too much
			sleep = 10 * time.Second
		}
	}
}

func ConfigureInterfaces() error {
	unix.Sethostname([]byte("ubmc"))

	// Fun story: if you don't have both IPv4 and IPv6 loopback configured
	// golang binaries will not bind to :: but to 0.0.0.0 instead.
	// Isn't that surprising?
	if err := addIp("127.0.0.1/8", "lo"); err != nil {
		return err
	}
	if err := addIp("::1/32", "lo"); err != nil {
		return err
	}
	if err := setLinkUp("lo"); err != nil {
		return err
	}

	iface := "eth0"
	if err := setLinkUp(iface); err != nil {
		return err
	}
	if err := addIp("10.0.10.20/24", iface); err != nil {
		return err
	}
	// If the MAC address changes on the interface the interface needs to be
	// taken down and up again in order for all IPv6 addresses and things to be
	// refreshed. MAC address changes happens when NC-SI reads the correct
	// MAC address from the adapter, or a controller hotswap potentially.
	go ipv6LinkFixer(iface)

	return nil
}
