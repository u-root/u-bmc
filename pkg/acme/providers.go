// Copyright 2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package acme

import (
	"context"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/libdns/cloudflare"
	"github.com/libdns/libdns"
)

// StubSolver dummy DNS01 solver
type StubSolver struct{}

// AppendRecords dummy function
func (s StubSolver) AppendRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	return recs, nil
}

// DeleteRecords dummy function
func (s StubSolver) DeleteRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	return make([]libdns.Record, 0), nil
}

// StubProvider returns a stub DNS01 challenge solver
func StubProvider() *certmagic.DNS01Solver {
	return &certmagic.DNS01Solver{
		DNSProvider:        StubSolver{},
		TTL:                time.Second,
		PropagationTimeout: time.Second,
	}
}

// CloudflareProvider returns a valid DNS01 challenge solver using Cloudflare DNS.
// Make sure to use a scoped API **token**, NOT a global API **key**. It will
// need two permissions: Zone-Zone-Read and Zone-DNS-Edit.
func CloudflareProvider(token string) *certmagic.DNS01Solver {
	return &certmagic.DNS01Solver{
		DNSProvider: &cloudflare.Provider{
			APIToken: token,
		},
	}
}
