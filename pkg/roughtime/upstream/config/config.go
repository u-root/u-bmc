// Copyright 2016 The Roughtime Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License. */

// Package config contains JSON structs for encoding information about
// Roughtime servers.
package config

// ServersJSON represents a JSON format for distributing information about
// Roughtime servers.
type ServersJSON struct {
	Servers []Server `json:"servers"`
}

// Server represents a Roughtime server in a JSON configuration.
type Server struct {
	Name string `json:"name"`
	// PublicKeyType specifies the type of the public key contained in
	// |PublicKey|. Normally this will be "ed25519" but implementations
	// should ignore entries with unknown key types.
	PublicKeyType string          `json:"publicKeyType"`
	PublicKey     []byte          `json:"publicKey"`
	Addresses     []ServerAddress `json:"addresses"`
}

// ServerAddress represents the address of a Roughtime server in a JSON
// configuration.
type ServerAddress struct {
	Protocol string `json:"protocol"`
	// Address contains a protocol specific address. For the protocol
	// "udp", the address has the form "host:port" where host is either a
	// DNS name, an IPv4 literal, or an IPv6 literal in square brackets.
	Address string `json:"address"`
}

// Chain represents a history of Roughtime queries where each provably follows
// the previous one.
type Chain struct {
	Links []Link `json:"links"`
}

// Link represents an entry in a Chain.
type Link struct {
	// PublicKeyType specifies the type of public key contained in
	// |PublicKey|. See the same field in |Server| for details.
	PublicKeyType string `json:"publicKeyType"`
	PublicKey     []byte `json:"serverPublicKey"`
	// NonceOrBlind contains either the full nonce (only for the first
	// |Link| in a |Chain|) or else contains a blind value that is combined
	// with the previous reply to make the next nonce. In either case, the
	// value is 64 bytes long.
	NonceOrBlind []byte `json:"nonceOrBlind"`
	// Reply contains the reply from the server.
	Reply []byte `json:"reply"`
}
