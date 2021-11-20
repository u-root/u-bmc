// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ttime

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/beevik/ntp"
	"github.com/cloudflare/roughtime"
	"github.com/cloudflare/roughtime/config"
	"github.com/u-root/u-bmc/pkg/service/logger"
	"golang.org/x/sync/errgroup"
)

var log = logger.LogContainer.GetSimpleLogger()

const (
	KEY_TYPE_ED25519   = "ed25519"
	ROUGHTIME_ATTEMPTS = 3
	ROUGHTIME_TIMEOUT  = 15 * time.Second
)

type RoughtimeServer struct {
	Protocol      string
	Address       string
	PublicKey     string
	PublicKeyType string
}

type NtpServer string

func getOneRoughtime(rs []RoughtimeServer) *roughtime.Roughtime {
	var g errgroup.Group
	cr := make(chan *roughtime.Roughtime, len(rs))
	for _, r := range rs {
		r := r
		pk, err := base64.StdEncoding.DecodeString(r.PublicKey)
		if err != nil {
			log.Warnf("Server %s has corrupt key (skipping): %v", r.Address, err)
			continue
		}
		srv := &config.Server{
			Name:          r.Address,
			PublicKeyType: r.PublicKeyType,
			PublicKey:     pk,
			Addresses: []config.ServerAddress{
				{Protocol: r.Protocol, Address: r.Address},
			}}
		g.Go(func() error {
			res, err := roughtime.Get(srv, ROUGHTIME_ATTEMPTS, ROUGHTIME_TIMEOUT, nil)
			if err != nil {
				log.Warnf("Failed to get roughtime from %s (skipping): %v", r.Address, err)
				return err
			}
			cr <- res
			return nil
		})
	}

	go func() {
		// In the case of an error, put a nil in on the queue to ensure that
		// this function never deadlocks
		if err := g.Wait(); err != nil {
			cr <- nil
		}
	}()

	return <-cr
}

func (n NtpServer) Server() string {
	return string(n)
}

func AcquireTime(rs []RoughtimeServer, ntps []NtpServer) (*time.Time, error) {
	// Calculate what the NTP servers would have reported at this time
	start := time.Now()
	rt := getOneRoughtime(rs)
	if rt == nil {
		return nil, fmt.Errorf("no roughtime servers available")
	}
	midpoint := rt.Midpoint.Unix()
	radius := time.Duration(rt.Radius) * time.Microsecond
	log.Infof("Acquired roughtime at %s (+/- %s)", midpoint.String(), radius.String())

	earliest := midpoint.Add(radius * -1)
	latest := midpoint.Add(radius)
	for _, n := range ntps {
		t, err := ntp.Time(n.Server())
		if err != nil {
			log.Warnf("Failed to contact NTP server %s (skipping): %v", n, err)
			continue
		}
		diff := time.Since(start)
		// Rewind timestamp to when the roughtime data was supposed to be valid
		ct := t.Add(diff * -1)
		if ct.After(latest) {
			log.Warnf("Rejecting bad NTP time from %s (%s > %s), it's too late", n, ct.String(), latest.String())
			continue
		}
		if t.Before(earliest) {
			log.Warnf("Rejecting bad NTP time from %s (%s < %s), it's too early", n, t.String(), earliest.String())
			continue
		}
		// Accept the first NTP time that inside the roughtime window
		log.Infof("NTP adjusted time to %s", t)
		return &t, nil
	}

	// Fall back to the roughtime time if no NTP servers are available
	return &midpoint, nil
}
