// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"net"
	"time"

	"github.com/mdlayher/genetlink"
	"github.com/mdlayher/netlink"
	"golang.org/x/sys/unix"
)

func registerNcsiPackage(b []byte) error {
	ad, err := netlink.NewAttributeDecoder(b)
	if err != nil {
		return err
	}

	for ad.Next() {
		switch ad.Type() {
		case unix.NCSI_PKG_ATTR_ID:
			id := ad.Uint32()
			log.Infof("NCSI package %d present", id)
		case unix.NCSI_PKG_ATTR_CHANNEL_LIST:
			ad.Do(handleNcsiChannelList)
		}
	}

	return ad.Err()
}

func registerNcsiChannel(b []byte) error {
	ad, err := netlink.NewAttributeDecoder(b)
	if err != nil {
		return err
	}

	var (
		id     = -1
		ls     = -1
		active = false
		forced = false
	)

	for ad.Next() {
		switch ad.Type() {
		case unix.NCSI_CHANNEL_ATTR_ID:
			id = int(ad.Uint32())
		case unix.NCSI_CHANNEL_ATTR_ACTIVE:
			active = true
		case unix.NCSI_CHANNEL_ATTR_FORCED:
			forced = true
		case unix.NCSI_CHANNEL_ATTR_LINK_STATE:
			ls = int(ad.Uint32())
		}
	}

	if err := ad.Err(); err != nil {
		return err
	}

	log.Infof("NCSI channel %d present [link state: %d, active: %v, forced: %v]", id, ls, active, forced)
	return nil
}

func handleNcsiChannelList(b []byte) error {
	ad, err := netlink.NewAttributeDecoder(b)
	if err != nil {
		return err
	}

	for ad.Next() {
		if ad.Type() == unix.NCSI_CHANNEL_ATTR {
			ad.Do(registerNcsiChannel)
		}
	}

	return ad.Err()
}

func handleNcsiPackageList(b []byte) error {
	ad, err := netlink.NewAttributeDecoder(b)
	if err != nil {
		return err
	}

	for ad.Next() {
		if ad.Type() == unix.NCSI_PKG_ATTR {
			ad.Do(registerNcsiPackage)
		}
	}

	return ad.Err()
}

func StartNcsi(iface string) {
	c, err := genetlink.Dial(nil)
	if err != nil {
		log.Errorf("dial generic netlink: %v", err)
		return
	}
	defer c.Close()

	family, err := c.GetFamily("NCSI")
	if err != nil {
		log.Errorf("get NCSI netlink family: %v", err)
		return
	}

	ifi, err := net.InterfaceByName(iface)
	if err != nil {
		log.Errorf("unable to get interface %s: %v", iface, err)
		return
	}

	ae := netlink.NewAttributeEncoder()
	ae.Uint32(unix.NCSI_ATTR_IFINDEX, uint32(ifi.Index))
	ed, err := ae.Encode()
	if err != nil {
		log.Errorf("failed to encode NCSI attribute data: %v", err)
		return
	}

	time.Sleep(15 * time.Second)
	for {
		req := genetlink.Message{
			Header: genetlink.Header{
				Command: unix.NCSI_CMD_PKG_INFO,
				Version: family.Version,
			},
			Data: ed,
		}

		msgs, err := c.Execute(req, family.ID, netlink.Request|netlink.Dump)
		if err != nil {
			log.Errorf("execute NCSI dump: %v", err)
			return
		}

		log.Infof("got %v replies", len(msgs))

		for _, m := range msgs {
			ad, err := netlink.NewAttributeDecoder(m.Data)
			if err != nil {
				log.Errorf("failed to create attribute decoder: %v", err)
				return
			}
			for ad.Next() {
				if ad.Type() == unix.NCSI_ATTR_PACKAGE_LIST {
					ad.Do(handleNcsiPackageList)
				}
			}

			if ad.Err() != nil {
				log.Errorf("failed to decode NCSI attributes: %v", err)
				return
			}
		}
		// TODO(bluecmd): We will only do this once for now
		// The idea is to have this as a GRPC call instead.
		break
		//time.Sleep(5 * time.Second)
	}
}
