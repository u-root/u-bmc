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

package protocol

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"testing"
	"testing/quick"

	"golang.org/x/crypto/ed25519"
)

func testEncodeDecodeRoundtrip(msg map[uint32][]byte) bool {
	encoded, err := Encode(msg)
	if err != nil {
		return true
	}

	decoded, err := Decode(encoded)
	if err != nil {
		return false
	}

	if len(msg) != len(decoded) {
		return false
	}

	for tag, payload := range msg {
		otherPayload, ok := decoded[tag]
		if !ok {
			return false
		}
		if !bytes.Equal(payload, otherPayload) {
			return false
		}
	}

	return true
}

func TestEncodeDecode(t *testing.T) {
	quick.Check(testEncodeDecodeRoundtrip, &quick.Config{
		MaxCountScale: 10,
	})
}

func TestRequestSize(t *testing.T) {
	_, _, request, err := CreateRequest(rand.Reader, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(request) != MinRequestSize {
		t.Errorf("got %d byte request, want %d bytes", len(request), MinRequestSize)
	}
}

func createServerIdentity(t *testing.T) (cert, rootPublicKey, onlinePrivateKey []byte) {
	rootPublicKey, rootPrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	onlinePublicKey, onlinePrivateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	if cert, err = CreateCertificate(0, 100, onlinePublicKey, rootPrivateKey); err != nil {
		t.Fatal(err)
	}

	return cert, rootPublicKey, onlinePrivateKey
}

func TestRoundtrip(t *testing.T) {
	cert, rootPublicKey, onlinePrivateKey := createServerIdentity(t)

	for _, numRequests := range []int{1, 2, 3, 4, 5, 15, 16, 17} {
		nonces := make([][NonceSize]byte, numRequests)
		for i := range nonces {
			binary.LittleEndian.PutUint32(nonces[i][:], uint32(i))
		}

		noncesSlice := make([][]byte, 0, numRequests)
		for i := range nonces {
			noncesSlice = append(noncesSlice, nonces[i][:])
		}

		const (
			expectedMidpoint = 50
			expectedRadius   = 5
		)

		replies, err := CreateReplies(noncesSlice, expectedMidpoint, expectedRadius, cert, onlinePrivateKey)
		if err != nil {
			t.Fatal(err)
		}

		for i, reply := range replies {
			midpoint, radius, err := VerifyReply(reply, rootPublicKey, nonces[i])
			if err != nil {
				t.Errorf("error parsing reply #%d: %s", i, err)
				continue
			}

			if midpoint != expectedMidpoint {
				t.Errorf("reply #%d gave a midpoint of %d, want %d", i, midpoint, expectedMidpoint)
			}
			if radius != expectedRadius {
				t.Errorf("reply #%d gave a radius of %d, want %d", i, radius, expectedRadius)
			}
		}
	}
}

func TestChaining(t *testing.T) {
	// This test demonstrates how a claim of misbehaviour from a client
	// would be checked. The client creates a two element chain in this
	// example where the first server says that the time is 10 and the
	// second says that it's 5.
	certA, rootPublicKeyA, onlinePrivateKeyA := createServerIdentity(t)
	certB, rootPublicKeyB, onlinePrivateKeyB := createServerIdentity(t)

	nonce1, _, _, err := CreateRequest(rand.Reader, nil)
	if err != nil {
		t.Fatal(err)
	}

	replies1, err := CreateReplies([][]byte{nonce1[:]}, 10, 0, certA, onlinePrivateKeyA)
	if err != nil {
		t.Fatal(err)
	}

	nonce2, blind2, _, err := CreateRequest(rand.Reader, replies1[0])
	if err != nil {
		t.Fatal(err)
	}

	replies2, err := CreateReplies([][]byte{nonce2[:]}, 5, 0, certB, onlinePrivateKeyB)
	if err != nil {
		t.Fatal(err)
	}

	// The client would present a series of tuples of (server identity,
	// nonce/blind, reply) as its claim of misbehaviour. The first element
	// contains a nonce where as all other elements contain just the
	// blinding value, as the nonce used for that request is calculated
	// from that and the previous reply.
	type claimStep struct {
		serverPublicKey []byte
		nonceOrBlind    [NonceSize]byte
		reply           []byte
	}

	claim := []claimStep{
		{rootPublicKeyA, nonce1, replies1[0]},
		{rootPublicKeyB, blind2, replies2[0]},
	}

	// In order to verify a claim, one would check each of the replies
	// based on the calculated nonce.
	var lastMidpoint uint64
	var misbehaviourFound bool
	for i, step := range claim {
		var nonce [NonceSize]byte
		if i == 0 {
			copy(nonce[:], step.nonceOrBlind[:])
		} else {
			nonce = CalculateChainNonce(claim[i-1].reply, step.nonceOrBlind[:])
		}
		midpoint, _, err := VerifyReply(step.reply, step.serverPublicKey, nonce)
		if err != nil {
			t.Fatal(err)
		}

		// This example doesn't take the radius into account.
		if i > 0 && midpoint < lastMidpoint {
			misbehaviourFound = true
		}
		lastMidpoint = midpoint
	}

	if !misbehaviourFound {
		t.Error("did not find expected misbehaviour")
	}
}
