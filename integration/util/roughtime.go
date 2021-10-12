// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net"

	"github.com/cloudflare/roughtime/mjd"
	"github.com/cloudflare/roughtime/protocol"
	"github.com/u-root/u-bmc/pkg/bmc/ttime"
	"golang.org/x/crypto/ed25519"
)

type RoughtimeServer struct {
	Config ttime.RoughtimeServer
	cert   []byte
	pk     []byte
}

func NewTestRoughtimeServer() *RoughtimeServer {
	rootPublicKey, rootPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate root key: %v", err)
	}

	onlinePublicKey, onlinePrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate online key: %v", err)
	}

	minTime := mjd.New(0, 0)
	maxTime := mjd.New(^uint64(0), 0)
	cert, err := protocol.CreateCertificate(minTime, maxTime, onlinePublicKey, rootPrivateKey)
	if err != nil {
		log.Fatalf("Failed to generate certificate: %v", err)
	}
	return &RoughtimeServer{
		Config: ttime.RoughtimeServer{
			Protocol:      "udp6",
			Address:       "[::1]:2002",
			PublicKey:     base64.StdEncoding.EncodeToString(rootPublicKey),
			PublicKeyType: ttime.KEY_TYPE_ED25519,
		},
		cert: cert,
		pk:   onlinePrivateKey,
	}
}

func (s *RoughtimeServer) Run() {
	var packetBuf [protocol.MinRequestSize]byte
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv6loopback, Port: 2002})
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	for {
		n, sourceAddr, err := conn.ReadFromUDP(packetBuf[:])
		if err != nil {
			log.Print(err)
		}
		if n < protocol.MinRequestSize {
			continue
		}
		packet, err := protocol.Decode(packetBuf[:n])
		if err != nil {
			continue
		}
		nonce, ok := packet[protocol.TagNonce]
		if !ok || len(nonce) != protocol.NonceSize {
			continue
		}
		midpoint := mjd.Now()
		radius := uint32(1000000)
		replies, err := protocol.CreateReplies([][]byte{nonce}, midpoint, radius, s.cert, s.pk)
		if err != nil {
			log.Print(err)
			continue
		}
		if len(replies) != 1 {
			continue
		}
		conn.WriteToUDP(replies[0], sourceAddr)
	}
}
