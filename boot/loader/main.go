// Copyright 2016-2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// loader mounts the root file system on /mnt/ and then
// validates the signature of /init against a build-in public key.
// If everything checks out /init is exec'd after a chroot.

package main

import (
	"context"
	"crypto"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/machinebox/progress"
	"golang.org/x/crypto/openpgp/errors"
	"golang.org/x/crypto/openpgp/packet"
	"golang.org/x/sys/unix"
)

const (
	pubKeyPath  = "/u-bmc.pub"
	initPath    = "/init"
	initSigPath = "/init.sig"
)

func main() {
	keyf, err := os.Open(pubKeyPath)
	if err != nil {
		log.Fatalf("Open(%s): %v", pubKeyPath, err)
	}
	err = unix.Mkdir("/mnt/", 0755)
	if err != nil {
		log.Fatalf("Mkdir(/mnt/): %v", err)
	}
	err = unix.Mount("ubi0:root", "/mnt", "ubifs", 0, "")
	if err != nil {
		log.Fatalf("Mount(ubi0:root): %v", err)
	}
	_ = unix.Mkdir("/mnt/boot", 0700)
	err = unix.Mount("ubi0:boot", "/mnt/boot", "ubifs", 0, "")
	if err != nil {
		log.Printf("Mount(ubi0:boot): %v", err)
	}
	err = unix.Chroot("/mnt/")
	if err != nil {
		log.Fatalf("chroot: %v", err)
	}
	sigf, err := os.Open(initSigPath)
	if err != nil {
		log.Fatalf("Open(%s): %v", initSigPath, err)
	}
	contentf, err := os.Open(initPath)
	if err != nil {
		log.Fatalf("Open(%s): %v", initPath, err)
	}
	key, err := readPublicSigningKey(keyf)
	if err != nil {
		log.Fatalf("readPublicSigningKey: %v", err)
	}
	if err = verifyDetachedSignature(key, contentf, sigf); err != nil {
		log.Fatalf("verify: %v", err)
	}
	log.Printf("Integrity check OK")
	err = unix.Exec("/init", []string{"/init"}, os.Environ())
	// This is only reached if Exec somehow failed
	log.Fatalf("exec: %v", err)
}

func readPublicSigningKey(keyf io.Reader) (*packet.PublicKey, error) {
	keypackets := packet.NewReader(keyf)
	p, err := keypackets.Next()
	if err != nil {
		return nil, err
	}
	switch pkt := p.(type) {
	case *packet.PublicKey:
		return pkt, nil
	default:
		log.Printf("ReadPublicSigningKey: got %T, want *packet.PublicKey", pkt)
	}
	return nil, errors.StructuralError("expected first packet to be PublicKey")
}

func verifyDetachedSignature(key *packet.PublicKey, contentf, sigf *os.File) error {
	var hashFunc crypto.Hash

	packets := packet.NewReader(sigf)
	p, err := packets.Next()
	if err != nil {
		return fmt.Errorf("reading signature file: %v", err)
	}
	switch sig := p.(type) {
	case *packet.Signature:
		hashFunc = sig.Hash
	case *packet.SignatureV3:
		hashFunc = sig.Hash
	default:
		return errors.UnsupportedError("unrecognized signature")
	}

	size, err := contentf.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("seek end: %v", err)
	}
	if _, err := contentf.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("seek start: %v", err)
	}

	r := progress.NewReader(contentf)

	go func() {
		ctx := context.Background()
		progressChan := progress.NewTicker(ctx, r, size, 200 * time.Millisecond)
		for p := range progressChan {
			fmt.Printf("Verifying /init integrity: %d %%\r", int(p.Percent()))
			os.Stdout.Sync()
		}
		fmt.Printf("Verifying /init integrity: complete\n")
	}()


	h := hashFunc.New()
	if _, err := io.Copy(h, r); err != nil && err != io.EOF {
		return err
	}
	switch sig := p.(type) {
	case *packet.Signature:
		err = key.VerifySignature(h, sig)
	case *packet.SignatureV3:
		err = key.VerifySignatureV3(h, sig)
	default:
		panic("unreachable")
	}
	return err
}
