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

// Package protocol implements the core of the Roughtime protocol.
package protocol

import (
	"bytes"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"sort"

	"golang.org/x/crypto/ed25519"
)

const (
	// NonceSize is the number of bytes in a nonce.
	NonceSize = sha512.Size
	// MinRequestSize is the minimum number of bytes in a request.
	MinRequestSize = 1024

	certificateContext    = "RoughTime v1 delegation signature--\x00"
	signedResponseContext = "RoughTime v1 response signature\x00"
)

// makeTag converts a four character string into a Roughtime tag value.
func makeTag(tag string) uint32 {
	if len(tag) != 4 {
		panic("makeTag: len(tag) != 4: " + tag)
	}

	return uint32(tag[0]) | uint32(tag[1])<<8 | uint32(tag[2])<<16 | uint32(tag[3])<<24
}

var (
	// Various tags used in the Roughtime protocol.
	tagCERT = makeTag("CERT")
	tagDELE = makeTag("DELE")
	tagINDX = makeTag("INDX")
	tagMAXT = makeTag("MAXT")
	tagMIDP = makeTag("MIDP")
	tagMINT = makeTag("MINT")
	tagNONC = makeTag("NONC")
	tagPAD  = makeTag("PAD\xff")
	tagPATH = makeTag("PATH")
	tagPUBK = makeTag("PUBK")
	tagRADI = makeTag("RADI")
	tagROOT = makeTag("ROOT")
	tagSIG  = makeTag("SIG\x00")
	tagSREP = makeTag("SREP")

	// TagNonce names the bytestring containing the client's nonce.
	TagNonce = tagNONC
)

// tagsSlice is the type of an array of tags. It provides utility functions so
// that they can be sorted.
type tagsSlice []uint32

func (t tagsSlice) Len() int           { return len(t) }
func (t tagsSlice) Less(i, j int) bool { return t[i] < t[j] }
func (t tagsSlice) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }

// Encode converts a map of tags to bytestrings into an encoded message. The
// number of elements in msg and the sum of the lengths of all the bytestrings
// must be ≤ 2**32.
func Encode(msg map[uint32][]byte) ([]byte, error) {
	if len(msg) == 0 {
		return make([]byte, 4), nil
	}

	if len(msg) >= math.MaxInt32 {
		return nil, errors.New("encode: too many tags")
	}

	var payloadSum uint64
	for _, payload := range msg {
		if len(payload)%4 != 0 {
			return nil, errors.New("encode: length of value is not a multiple of four")
		}
		payloadSum += uint64(len(payload))
	}
	if payloadSum >= 1<<32 {
		return nil, errors.New("encode: payloads too large")
	}

	tags := tagsSlice(make([]uint32, 0, len(msg)))
	for tag := range msg {
		tags = append(tags, tag)
	}
	sort.Sort(tags)

	numTags := uint64(len(tags))

	encoded := make([]byte, 4*(1+numTags-1+numTags)+payloadSum)
	binary.LittleEndian.PutUint32(encoded, uint32(len(tags)))
	offsets := encoded[4:]
	tagBytes := encoded[4*(1+(numTags-1)):]
	payloads := encoded[4*(1+(numTags-1)+numTags):]

	currentOffset := uint32(0)

	for i, tag := range tags {
		payload := msg[tag]
		if i > 0 {
			binary.LittleEndian.PutUint32(offsets, currentOffset)
			offsets = offsets[4:]
		}

		binary.LittleEndian.PutUint32(tagBytes, tag)
		tagBytes = tagBytes[4:]

		if len(payload) > 0 {
			copy(payloads, payload)
			payloads = payloads[len(payload):]
			currentOffset += uint32(len(payload))
		}
	}

	return encoded, nil
}

// Decode parses the output of encode back into a map of tags to bytestrings.
func Decode(bytes []byte) (map[uint32][]byte, error) {
	if len(bytes) < 4 {
		return nil, errors.New("decode: message too short to be valid")
	}
	if len(bytes)%4 != 0 {
		return nil, errors.New("decode: message is not a multiple of four bytes")
	}

	numTags := uint64(binary.LittleEndian.Uint32(bytes))

	if numTags == 0 {
		return make(map[uint32][]byte), nil
	}

	minLen := 4 * (1 + (numTags - 1) + numTags)

	if uint64(len(bytes)) < minLen {
		return nil, errors.New("decode: message too short to be valid")
	}

	offsets := bytes[4:]
	tags := bytes[4*(1+numTags-1):]
	payloads := bytes[minLen:]

	if len(payloads) > math.MaxInt32 {
		return nil, errors.New("decode: message too large")
	}
	payloadLength := uint32(len(payloads))

	currentOffset := uint32(0)
	var lastTag uint32
	ret := make(map[uint32][]byte)

	for i := uint64(0); i < numTags; i++ {
		tag := binary.LittleEndian.Uint32(tags)
		tags = tags[4:]

		if i > 0 && lastTag >= tag {
			return nil, errors.New("decode: tags out of order")
		}

		var nextOffset uint32
		if i < numTags-1 {
			nextOffset = binary.LittleEndian.Uint32(offsets)
			offsets = offsets[4:]
		} else {
			nextOffset = payloadLength
		}

		if nextOffset%4 != 0 {
			return nil, errors.New("decode: payload length is not a multiple of four bytes")
		}

		if nextOffset < currentOffset {
			return nil, errors.New("decode: offsets out of order")
		}

		length := nextOffset - currentOffset
		if uint32(len(payloads)) < length {
			return nil, errors.New("decode: message truncated")
		}

		payload := payloads[:length]
		payloads = payloads[length:]
		ret[tag] = payload
		currentOffset = nextOffset
	}

	return ret, nil
}

// messageOverhead returns the number of bytes needed for Encode to encode the
// given number of tags.
func messageOverhead(numTags int) int {
	return 4 * 2 * numTags
}

// CalculateChainNonce calculates the nonce to be used in the next request in a
// chain given a reply and a blinding factor.
func CalculateChainNonce(prevReply, blind []byte) (nonce [NonceSize]byte) {
	h := sha512.New()
	h.Write(prevReply)
	prevReplyHash := h.Sum(nil)

	h.Reset()
	h.Write(prevReplyHash)
	h.Write(blind)
	h.Sum(nonce[:0])

	return nonce
}

// CreateRequest creates a Roughtime request given an entropy source and the
// contents of a previous reply for chaining. If this request is the first of a
// chain, prevReply can be empty. It returns the nonce (needed to verify the
// reply), the blind (needed to prove correct chaining to an external party)
// and the request itself.
func CreateRequest(rand io.Reader, prevReply []byte) (nonce, blind [NonceSize]byte, request []byte, err error) {
	if _, err := io.ReadFull(rand, blind[:]); err != nil {
		return nonce, blind, nil, err
	}

	nonce = CalculateChainNonce(prevReply, blind[:])

	padding := make([]byte, MinRequestSize-messageOverhead(2)-len(nonce))
	msg, err := Encode(map[uint32][]byte{
		tagNONC: nonce[:],
		tagPAD:  padding,
	})
	if err != nil {
		return nonce, blind, nil, err
	}

	return nonce, blind, msg, nil
}

// tree represents a Merkle tree of nonces. Each element of values is a layer
// in the tree, with the widest layer first.
type tree struct {
	values [][][NonceSize]byte
}

var (
	hashLeafTweak = []byte{0}
	hashNodeTweak = []byte{1}
)

// hashLeaf hashes an nonce to form the leaf of the Merkle tree.
func hashLeaf(out *[sha512.Size]byte, in []byte) {
	h := sha512.New()
	h.Write(hashLeafTweak)
	h.Write(in)
	h.Sum(out[:0])
}

// hashNode hashes two child elements of the Merkle tree to produce an interior
// node.
func hashNode(out *[sha512.Size]byte, left, right []byte) {
	h := sha512.New()
	h.Write(hashNodeTweak)
	h.Write(left)
	h.Write(right)
	h.Sum(out[:0])
}

// newTree creates a Merkle tree given one or more nonces.
func newTree(nonces [][]byte) *tree {
	if len(nonces) == 0 {
		panic("newTree: passed empty slice")
	}

	levels := 1
	width := len(nonces)
	for width > 1 {
		width = (width + 1) / 2
		levels++
	}

	ret := &tree{
		values: make([][][NonceSize]byte, 0, levels),
	}

	leaves := make([][NonceSize]byte, ((len(nonces)+1)/2)*2)
	for i, nonce := range nonces {
		var leaf [NonceSize]byte
		hashLeaf(&leaf, nonce)
		leaves[i] = leaf
	}
	ret.values = append(ret.values, leaves)

	for i := 1; i < levels; i++ {
		lastLevel := ret.values[i-1]
		width := len(lastLevel) / 2
		if width%2 == 1 {
			width++
		}
		level := make([][NonceSize]byte, width)
		for j := 0; j < len(lastLevel)/2; j++ {
			hashNode(&level[j], lastLevel[j*2][:], lastLevel[j*2+1][:])
		}
		ret.values = append(ret.values, level)
	}

	return ret
}

// Root returns the root value of t.
func (t *tree) Root() *[NonceSize]byte {
	return &t.values[len(t.values)-1][0]
}

// Levels returns the number of levels in t.
func (t *tree) Levels() int {
	return len(t.values)
}

// Path returns elements from t needed to prove, given the root, that the leaf
// at the given index is in the tree.
func (t *tree) Path(index int) (path [][]byte) {
	path = make([][]byte, 0, len(t.values))

	for level := 0; level < len(t.values)-1; level++ {
		if index%2 == 1 {
			path = append(path, t.values[level][index-1][:])
		} else {
			path = append(path, t.values[level][index+1][:])
		}

		index /= 2
	}

	return path
}

// CreateReplies signs, using privateKey, a batch of nonces along with the
// given time and radius in microseconds. It returns one reply for each nonce
// using that signature and includes cert in each.
func CreateReplies(nonces [][]byte, midpoint uint64, radius uint32, cert []byte, privateKey []byte) ([][]byte, error) {
	if len(nonces) == 0 {
		return nil, nil
	}

	tree := newTree(nonces)

	var midpointBytes [8]byte
	binary.LittleEndian.PutUint64(midpointBytes[:], midpoint)
	var radiusBytes [4]byte
	binary.LittleEndian.PutUint32(radiusBytes[:], radius)

	signedReply := map[uint32][]byte{
		tagMIDP: midpointBytes[:],
		tagRADI: radiusBytes[:],
		tagROOT: tree.Root()[:],
	}
	signedReplyBytes, err := Encode(signedReply)
	if err != nil {
		return nil, err
	}

	toBeSigned := signedResponseContext + string(signedReplyBytes)
	sig := ed25519.Sign(privateKey, []byte(toBeSigned))

	reply := map[uint32][]byte{
		tagSREP: signedReplyBytes,
		tagSIG:  sig,
		tagCERT: cert,
	}

	replies := make([][]byte, 0, len(nonces))

	for i := range nonces {
		var indexBytes [4]byte
		binary.LittleEndian.PutUint32(indexBytes[:], uint32(i))
		reply[tagINDX] = indexBytes[:]

		path := tree.Path(i)
		pathBytes := make([]byte, 0, NonceSize*len(path))
		for _, pathStep := range path {
			pathBytes = append(pathBytes, pathStep...)
		}
		reply[tagPATH] = pathBytes

		replyBytes, err := Encode(reply)
		if err != nil {
			return nil, err
		}

		replies = append(replies, replyBytes)
	}

	return replies, nil
}

// CreateCertificate returns a signed certificate, using rootPrivateKey,
// delegating authority for the given timestamp to publicKey.
func CreateCertificate(minTime, maxTime uint64, publicKey, rootPrivateKey []byte) (certBytes []byte, err error) {
	if maxTime < minTime {
		return nil, errors.New("protocol: maxTime < minTime")
	}

	var minTimeBytes, maxTimeBytes [8]byte
	binary.LittleEndian.PutUint64(minTimeBytes[:], minTime)
	binary.LittleEndian.PutUint64(maxTimeBytes[:], maxTime)

	signed := map[uint32][]byte{
		tagPUBK: publicKey,
		tagMINT: minTimeBytes[:],
		tagMAXT: maxTimeBytes[:],
	}

	signedBytes, err := Encode(signed)
	if err != nil {
		return nil, err
	}

	toBeSigned := certificateContext + string(signedBytes)
	sig := ed25519.Sign(rootPrivateKey, []byte(toBeSigned))

	cert := map[uint32][]byte{
		tagSIG:  sig,
		tagDELE: signedBytes,
	}

	return Encode(cert)
}

func getValue(msg map[uint32][]byte, tag uint32, name string) (value []byte, err error) {
	value, ok := msg[tag]
	if !ok {
		return nil, errors.New("protocol: missing " + name)
	}
	return value, nil
}

func getFixedLength(msg map[uint32][]byte, tag uint32, name string, length int) (value []byte, err error) {
	value, err = getValue(msg, tag, name)
	if err != nil {
		return nil, err
	}
	if len(value) != length {
		return nil, errors.New("protocol: incorrect length for " + name)
	}
	return value, nil
}

func getUint32(msg map[uint32][]byte, tag uint32, name string) (result uint32, err error) {
	valueBytes, err := getFixedLength(msg, tag, name, 4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(valueBytes), nil
}

func getUint64(msg map[uint32][]byte, tag uint32, name string) (result uint64, err error) {
	valueBytes, err := getFixedLength(msg, tag, name, 8)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(valueBytes), nil
}

func getSubmessage(msg map[uint32][]byte, tag uint32, name string) (result map[uint32][]byte, err error) {
	valueBytes, err := getValue(msg, tag, name)
	if err != nil {
		return nil, err
	}

	result, err = Decode(valueBytes)
	if err != nil {
		return nil, errors.New("protocol: failed to parse " + name + ": " + err.Error())
	}

	return result, nil
}

// VerifyReply parses the Roughtime reply in replyBytes, authenticates it using
// publicKey and verifies that nonce is included in it. It returns the included
// timestamp and radius.
func VerifyReply(replyBytes, publicKey []byte, nonce [NonceSize]byte) (time uint64, radius uint32, err error) {
	reply, err := Decode(replyBytes)
	if err != nil {
		return 0, 0, errors.New("protocol: failed to parse top-level reply: " + err.Error())
	}

	cert, err := getSubmessage(reply, tagCERT, "certificate")
	if err != nil {
		return 0, 0, err
	}

	signatureBytes, err := getFixedLength(cert, tagSIG, "signature", ed25519.SignatureSize)
	if err != nil {
		return 0, 0, err
	}

	delegationBytes, err := getValue(cert, tagDELE, "delegation")
	if err != nil {
		return 0, 0, err
	}

	if !ed25519.Verify(publicKey, []byte(certificateContext+string(delegationBytes)), signatureBytes) {
		return 0, 0, errors.New("protocol: invalid delegation signature")
	}

	delegation, err := Decode(delegationBytes)
	if err != nil {
		return 0, 0, errors.New("protocol: failed to parse delegation: " + err.Error())
	}

	minTime, err := getUint64(delegation, tagMINT, "minimum time")
	if err != nil {
		return 0, 0, err
	}

	maxTime, err := getUint64(delegation, tagMAXT, "maximum time")
	if err != nil {
		return 0, 0, err
	}

	delegatedPublicKey, err := getFixedLength(delegation, tagPUBK, "public key", ed25519.PublicKeySize)
	if err != nil {
		return 0, 0, err
	}

	responseSigBytes, err := getFixedLength(reply, tagSIG, "signature", ed25519.SignatureSize)
	if err != nil {
		return 0, 0, err
	}

	signedResponseBytes, ok := reply[tagSREP]
	if !ok {
		return 0, 0, errors.New("protocol: response is missing signed portion")
	}

	if !ed25519.Verify(delegatedPublicKey, []byte(signedResponseContext+string(signedResponseBytes)), responseSigBytes) {
		return 0, 0, errors.New("protocol: invalid response signature")
	}

	signedResponse, err := Decode(signedResponseBytes)
	if err != nil {
		return 0, 0, errors.New("protocol: failed to parse signed response: " + err.Error())
	}

	root, err := getFixedLength(signedResponse, tagROOT, "root", sha512.Size)
	if err != nil {
		return 0, 0, err
	}

	midpoint, err := getUint64(signedResponse, tagMIDP, "midpoint")
	if err != nil {
		return 0, 0, err
	}

	radius, err = getUint32(signedResponse, tagRADI, "radius")
	if err != nil {
		return 0, 0, err
	}

	if maxTime < minTime {
		return 0, 0, errors.New("protocol: invalid delegation range")
	}

	if midpoint < minTime || maxTime < midpoint {
		return 0, 0, errors.New("protocol: timestamp out of range for delegation")
	}

	index, err := getUint32(reply, tagINDX, "index")
	if err != nil {
		return 0, 0, err
	}

	path, err := getValue(reply, tagPATH, "path")
	if err != nil {
		return 0, 0, err
	}
	if len(path)%sha512.Size != 0 {
		return 0, 0, errors.New("protocol: path is not a multiple of the hash size")
	}

	var hash [sha512.Size]byte
	hashLeaf(&hash, nonce[:])

	for len(path) > 0 {
		pathElementIsRight := index&1 == 0
		if pathElementIsRight {
			hashNode(&hash, hash[:], path[:sha512.Size])
		} else {
			hashNode(&hash, path[:sha512.Size], hash[:])
		}

		index >>= 1
		path = path[sha512.Size:]
	}

	if !bytes.Equal(hash[:], root) {
		return 0, 0, errors.New("protocol: calculated tree root doesn't match signed root")
	}

	return midpoint, radius, nil
}
