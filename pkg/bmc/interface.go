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

	pb "github.com/u-root/u-bmc/proto"
	"github.com/u-root/u-root/pkg/dhclient"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

const (
	interfaceUpTimeout = 30 * time.Second
)

type network struct {
	fqdn string
	ipv4 net.IP
	ipv6 net.IP
}

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

func (n *network) FQDN() string {
	return n.fqdn
}

func (n *network) IPv4() net.IP {
	return n.ipv4
}

func (n *network) IPv6() net.IP {
	return n.ipv4
}

func (n *network) AddressLifetime() time.Duration {
	// TODO(bluecmd): This should be decided based on DHCP lease and such
	return time.Hour
}

func startNetwork(config *pb.Network) (*network, error) {
	if config == nil {
		log.Printf("No network configuration detected, using defaults")
		config = &pb.Network{}
	}

	// Fun story: if you don't have both IPv4 and IPv6 loopback configured
	// golang binaries will not bind to :: but to 0.0.0.0 instead.
	// Isn't that surprising?
	if err := addIp("127.0.0.1/8", "lo"); err != nil {
		return nil, err
	}
	if err := addIp("::1/32", "lo"); err != nil {
		return nil, err
	}
	if err := setLinkUp("lo"); err != nil {
		return nil, err
	}

	iface := "eth0"

	// TODO(bluecmd): Set ipv4/ipv6 objects to remember the host addresses
	if config.Vlan != 0 {
		log.Printf("TODO: Interface was configured to use VLAN but that's not implemented yet")
	}

	_, err := dhclient.IfUp(iface)
	if err != nil {
		return nil, err
	}

	// If the MAC address changes on the interface the interface needs to be
	// taken down and up again in order for all IPv6 addresses and things to be
	// refreshed. MAC address changes happens when NC-SI reads the correct
	// MAC address from the adapter, or a controller hotswap potentially.
	go ipv6LinkFixer(iface)

	if config.Ipv4Address != "" {
		if err := addIp(config.Ipv4Address, iface); err != nil {
			log.Printf("Error adding IPv4 %s to interface %s: %v", config.Ipv4Address, iface, err)
		}
	}
	if config.Ipv6Address != "" {
		if err := addIp(config.Ipv6Address, iface); err != nil {
			log.Printf("Error adding IPv6 %s to interface %s: %v", config.Ipv6Address, iface, err)
		}
	}

	if len(config.Ipv4Route)+len(config.Ipv6Route) > 0 {
		log.Printf("TODO: IP routes are configured but not supported yet")
	}

	go func() {
		c := make(chan *RDNSSOption)
		go rdnss(c)
		for r := range c {
			log.Printf("TODO: got RDNSS %v", r)
		}
	}()

	// When we exit this function we must have received a hostname or otherwise
	// had one configured. The rest of the startup flow depends on it.

	// TODO(bluecmd): Read hostname from config file or DHCP, don't have any default
	fqdn := "ubmc.local"
	if config.Hostname != "" {
		fqdn = config.Hostname
	}
	unix.Sethostname([]byte(fqdn))

	return &network{fqdn: fqdn}, nil
}
