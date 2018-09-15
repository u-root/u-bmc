// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"time"

	"github.com/mdlayher/genetlink"
	"github.com/mdlayher/netlink"
	vnl "github.com/vishvananda/netlink"
)

const (
	NCSI_CMD_PKG_INFO        = 1
	NCSI_CMD_SET_INTERFACE   = 2
	NCSI_CMD_CLEAR_INTERFACE = 3

	NCSI_ATTR_IFINDEX      = 1
	NCSI_ATTR_PACKAGE_LIST = 2
	NCSI_ATTR_PACKAGE_ID   = 3
	NCSI_ATTR_CHANNEL_ID   = 4

	NCSI_PKG_ATTR              = 1
	NCSI_PKG_ATTR_ID           = 2
	NCSI_PKG_ATTR_FORCED       = 3
	NCSI_PKG_ATTR_CHANNEL_LIST = 4

	NCSI_CHANNEL_ATTR               = 1
	NCSI_CHANNEL_ATTR_ID            = 2
	NCSI_CHANNEL_ATTR_VERSION_MAJOR = 3
	NCSI_CHANNEL_ATTR_VERSION_MINOR = 4
	NCSI_CHANNEL_ATTR_VERSION_STR   = 5
	NCSI_CHANNEL_ATTR_LINK_STATE    = 6
	NCSI_CHANNEL_ATTR_ACTIVE        = 7
	NCSI_CHANNEL_ATTR_FORCED        = 8
	NCSI_CHANNEL_ATTR_VLAN_LIST     = 9
	NCSI_CHANNEL_ATTR_VLAN_ID       = 10
)

func registerNcsiPackage(b []byte) {
	ad, err := netlink.NewAttributeDecoder(b)
	if err != nil {
		log.Fatalf("failed to create attribute decoder: %v", err)
	}

	for ad.Next() {
		if ad.Type() == NCSI_PKG_ATTR_ID {
			id := ad.Uint32()
			log.Printf("NCSI package %d present", id)
		} else if ad.Type() == NCSI_PKG_ATTR_CHANNEL_LIST {
			handleNcsiChannelList(ad.Bytes())
		}
	}
}

func registerNcsiChannel(b []byte) {
	ad, err := netlink.NewAttributeDecoder(b)
	if err != nil {
		log.Fatalf("failed to create attribute decoder: %v", err)
	}

	id := -1
	ls := -1
	active := false
	forced := false
	for ad.Next() {
		if ad.Type() == NCSI_CHANNEL_ATTR_ID {
			id = int(ad.Uint32())
		} else if ad.Type() == NCSI_CHANNEL_ATTR_ACTIVE {
			active = true
		} else if ad.Type() == NCSI_CHANNEL_ATTR_FORCED {
			forced = true
		} else if ad.Type() == NCSI_CHANNEL_ATTR_LINK_STATE {
			ls = int(ad.Uint32())
		}
	}
	log.Printf("NCSI channel %d present [link state: %d, active: %v, forced: %v]", id, ls, active, forced)
}

func handleNcsiChannelList(b []byte) {
	ad, err := netlink.NewAttributeDecoder(b)
	if err != nil {
		log.Fatalf("failed to create attribute decoder: %v", err)
	}

	for ad.Next() {
		if ad.Type() == NCSI_CHANNEL_ATTR {
			registerNcsiChannel(ad.Bytes())
		}
	}
}

func handleNcsiPackageList(b []byte) {
	ad, err := netlink.NewAttributeDecoder(b)
	if err != nil {
		log.Fatalf("failed to create attribute decoder: %v", err)
	}

	for ad.Next() {
		if ad.Type() == NCSI_PKG_ATTR {
			registerNcsiPackage(ad.Bytes())
		}
	}
}

func startNcsi(iface string) {
	c, err := genetlink.Dial(nil)
	if err != nil {
		log.Fatalf("dial generic netlink: %v", err)
	}
	defer c.Close()

	family, err := c.GetFamily("NCSI")
	if err != nil {
		log.Fatalf("get NCSI netlink family: %v", err)
	}

	l, err := vnl.LinkByName(iface)
	if err != nil {
		log.Fatalf("unable to get interface %s: %v", iface, err)
	}

	ae := netlink.NewAttributeEncoder()
	ae.Uint32(NCSI_ATTR_IFINDEX, uint32(l.Attrs().Index))
	ed, err := ae.Encode()
	if err != nil {
		log.Fatalf("failed to encode NCSI attribute data: %v", err)
	}

	time.Sleep(15 * time.Second)
	for {
		req := genetlink.Message{
			Header: genetlink.Header{
				Command: NCSI_CMD_PKG_INFO,
				Version: family.Version,
			},
			Data: ed,
		}

		msgs, err := c.Execute(req, family.ID, netlink.HeaderFlagsRequest|netlink.HeaderFlagsDump)
		if err != nil {
			log.Fatalf("execute NCSI dump: %v", err)
		}

		log.Printf("got %v replies", len(msgs))

		for _, m := range msgs {
			ad, err := netlink.NewAttributeDecoder(m.Data)
			if err != nil {
				log.Fatalf("failed to create attribute decoder: %v", err)
			}
			for ad.Next() {
				if ad.Type() == NCSI_ATTR_PACKAGE_LIST {
					handleNcsiPackageList(ad.Bytes())
				}
			}
		}
		// TODO(bluecmd): We will only do this once for now
		// The idea is to have this as a GRPC call instead.
		break
		//time.Sleep(5 * time.Second)
	}
}
