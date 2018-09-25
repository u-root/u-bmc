// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/u-root/dhcp4/dhcp4client"
	"github.com/u-root/dhcp4/dhcp4opts"
	"github.com/u-root/u-root/pkg/dhclient"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

const (
	interfaceUpTimeout       = 30 * time.Second
	retryDelay               = 60 * time.Second
	retryDelaySecondsJitter  = 10
	defaultLeaseTime         = 7 * 24 * time.Hour
	dhcpTimeout              = 10 * time.Second
	dhcpRetry                = 3
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

func doRetryDelay() {
	delay := retryDelay + time.Duration(rand.Intn(retryDelaySecondsJitter)) * time.Second
	log.Printf("Waiting %v before retrying", delay)
	time.Sleep(delay)
}

func doDHCP4(iface string) {
	l, err := netlink.LinkByName(iface)
	if err != nil {
		// Permanent error
		log.Printf("Unable to get interface %s: %v", iface, err)
		return
	}
	client, err := dhcp4client.New(l, dhcp4client.WithTimeout(dhcpTimeout), dhcp4client.WithRetry(dhcpRetry))
	if err != nil {
		// Permanent error
		log.Printf("Failed to create DHCPv4 client for %s: %v", iface, err)
		return
	}
	// Initial acquisition is silent
	var packet *dhclient.Packet4
	for {
		log.Printf("DEBUG: client.Request")
		p, err := client.Request()
		if err == nil {
			packet = dhclient.NewPacket4(p)
			break
		}
		delay := retryDelay + time.Duration(rand.Intn(retryDelaySecondsJitter)) * time.Second
		log.Printf("DEBUG: sleep for %s, %v", delay.String(), err)
		time.Sleep(delay)
	}
	log.Printf("Acquired DHCPv4 lease on %s, IP: %s", iface, packet.Lease().String())
	leaseTime, err := dhcp4opts.GetIPAddressLeaseTime(packet.P.Options)
	if err != nil {
		// No lease time, use default
		leaseTime = defaultLeaseTime
	}
	// Renewal
	for {
		if packet != nil {
			dhclient.Configure4(l, packet.P)
		}
		delay := leaseTime + time.Duration(rand.Intn(retryDelaySecondsJitter)) * time.Second
		log.Printf("Renewing DHPCv4 lease in %s", delay.String())
		time.Sleep(delay)
		p, err := client.Renew(packet.P)
		if err != nil {
			log.Printf("Failed to renew DHPCv4 lease for %s: %v", iface, err)
			leaseTime = time.Second
		}
		packet = dhclient.NewPacket4(p)
	}
}

func doDHCP6(iface string) {
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

	// TODO(bluecmd): Read MAC address from NC-SI
	iface := "eth0"
	if err := setLinkUp(iface); err != nil {
		return err
	}
	go doDHCP4(iface)
	go doDHCP6(iface)

	return nil
}
