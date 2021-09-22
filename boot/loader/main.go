// Copyright 2016-2021 the u-root Authors. All rights reserved
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
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/machinebox/progress"
	"github.com/u-root/u-root/pkg/kmodule"
	"golang.org/x/crypto/openpgp/errors"
	"golang.org/x/crypto/openpgp/packet"
	"golang.org/x/sys/unix"
)

const (
	UUID       = "26ab0401-3f49-4fc2-a172-c8aa02aceaf3"
	pubKeyPath = "/u-bmc.pub"
	kernelPath = "/mnt/boot/zImage"
	dtbPath    = "/mnt/boot/platform.dtb"
	initPath   = "/mnt/bin/init"
)

var (
	mtd    = flag.Bool("mtd", false, "Mount and load u-bmc from MTD flash")
	blk    = flag.Bool("blk", false, "Mount and load u-bmc from block device")
	ast    = flag.Bool("ast", false, "ASPEED ast specific option")
	verify = []string{initPath, kernelPath, dtbPath}
)

func main() {
	flag.Parse()
	if *mtd && *blk {
		log.Fatal("Please choose either mtd or blk, not both!")
	}

	if *ast {
		err := loadModule("/bootlock.ko")
		if err != nil {
			log.Fatalf("loadModule(/bootlock.ko): %v", err)
		}
	}

	keyf, err := os.Open(pubKeyPath)
	if err != nil {
		log.Fatalf("Open(%s): %v", pubKeyPath, err)
	}
	key, err := readPublicSigningKey(keyf)
	if err != nil {
		log.Fatalf("readPublicSigningKey(%s): %v", pubKeyPath, err)
	}

	if *mtd {
		err = unix.Mount("ubi0:root", "/mnt", "ubifs", unix.MS_RDONLY, "")
		if err != nil {
			log.Fatalf("Mount(ubi0:root): %v", err)
		}
	}
	if *blk {
		err = unix.Mount("UUID="+UUID, "/ro", "erofs", unix.MS_RDONLY, "")
		if err != nil {
			log.Fatalf("Mount(%s): %v", "UUID="+UUID, err)
		}
	}

	for _, path := range verify {
		f, err := openAndVerify(path, key)
		if err != nil {
			log.Fatalf("openAndVerify(%s): %v", path, err)
		}
		f.Close()
	}
	log.Printf("Integrity check OK")

	// Load the runtime kernel
	// TODO(bluecmd): Use u-root kexec package when it supports ARM
	// https://github.com/u-root/u-root/issues/401
	cmd := exec.Command("/kexec", "-d", "-l", kernelPath, "--dtb", dtbPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatalf("cmd.Run(kexec -d -l %s --dtb %s): %v", kernelPath, dtbPath, err)
	}

	err = unix.Unmount("/mnt", 0)
	if err != nil {
		log.Fatalf("Unmount(/mnt): %v", err)
	}

	cmd = exec.Command("/kexec", "-e")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	// This is only reached if kexec somehow failed
	log.Fatalf("cmd.Run(kexec -e): %v", err)
}

func openAndVerify(path string, key *packet.PublicKey) (*os.File, error) {
	sigf, err := os.Open(path + ".gpg")
	if err != nil {
		return nil, err
	}
	defer sigf.Close()
	contentf, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	if err = verifyDetachedSignature(contentf, sigf, key); err != nil {
		return nil, err
	}
	return contentf, nil
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

func verifyDetachedSignature(contentf, sigf *os.File, key *packet.PublicKey) error {
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
	c := make(chan struct{})

	go func(path string) {
		ctx := context.Background()
		path, err := filepath.EvalSymlinks(path)
		if err != nil {
			path = fmt.Sprintf("{%v}", err)
		}
		progressChan := progress.NewTicker(ctx, r, size, 200*time.Millisecond)
		for p := range progressChan {
			fmt.Printf("Verifying %s integrity: %d %%\r", path, int(p.Percent()))
			os.Stdout.Sync()
		}
		fmt.Printf("Verifying %s integrity: complete\n", path)
		close(c)
	}(contentf.Name())

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
	// Wait for the final status printout to not mess up the log
	_ = <-c
	return err
}

func loadModule(fp string) error {
	f, err := os.Open(fp)
	if err != nil {
		return err
	}
	defer f.Close()
	return kmodule.FileInit(f, "", 0)
}
