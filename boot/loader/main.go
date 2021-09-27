// Copyright 2016-2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// loader can operate in two modes: kexec and switch
// In kexec mode it mounts the rootfs to /ro and
// validates the files in there against its keys.
// If everything matches it uses kexec to load and
// execute the previously validated kernel.
// In switch mode it just sets up the rootfs mount
// and spans an overlayfs on top. After that it uses
// switch_root to move the mount points and run init.

package main

import (
	"bytes"
	"context"
	"crypto"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/machinebox/progress"
	"github.com/u-root/u-root/pkg/boot"
	"github.com/u-root/u-root/pkg/boot/kexec"
	"github.com/u-root/u-root/pkg/kmodule"
	uroot "github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/uio"
	"golang.org/x/crypto/openpgp/errors"
	"golang.org/x/crypto/openpgp/packet"
	"golang.org/x/sys/unix"
)

const (
	pubKeyPath = "/u-bmc.pub"
	kernelPath = "/ro/boot/zImage"
	dtbPath    = "/ro/boot/platform.dtb"
	initPath   = "/ro/bin/init"
)

var (
	kload  = flag.Bool("kexec", false, "Mount rootfs and call kexec")
	swroot = flag.Bool("switch", false, "Mount rootfs and call switch_root")
	mtd    = flag.Bool("mtd", false, "Mount and load u-bmc from MTD flash")
	blk    = flag.Bool("blk", false, "Mount and load u-bmc from block device")
	ast    = flag.Bool("ast", false, "ASPEED ast specific option")
	verify = []string{initPath, kernelPath, dtbPath}
)

func main() {
	flag.Parse()
	if *mtd && *blk {
		log.Fatal("mtd and blk are mutually exclusive!")
	}
	if !*mtd && !*blk {
		log.Fatal("please choose either mtd or blk!")
	}
	if *kload && *swroot {
		log.Fatal("kexec and switch are mutually exclusive!")
	}
	if !*kload && !*swroot {
		log.Fatal("please choose either kexec or switch!")
	}
	if *ast {
		err := loadModule("/bootlock.ko")
		if err != nil {
			log.Fatalf("loadModule(/bootlock.ko): %v", err)
		}
	}
	if *kload {
		loadAndExec()
	}
	if *swroot {
		mountAndSwitchRoot()
	}
}

// mountAndSwitchRoot mounts the rootfs and overlay then runs switch_root
func mountAndSwitchRoot() {
	createBasicHirarchy()
	if *mtd {
		mountMtd()
	}
	if *blk {
		mountBlk()
	}
	mountOverlay()

	err := uroot.SwitchRoot("/mnt", "/bin/init")
	if err != nil {
		log.Fatalf("SwitchRoot: %v", err)
	}
}

// loadAndExec validates boot files and runs them via kexec
func loadAndExec() {
	keyf, err := os.Open(pubKeyPath)
	if err != nil {
		log.Fatalf("Open(%s): %v", pubKeyPath, err)
	}
	key, err := readPublicSigningKey(keyf)
	if err != nil {
		log.Fatalf("readPublicSigningKey(%s): %v", pubKeyPath, err)
	}

	createBasicHirarchy()
	if *mtd {
		mountMtd()
	}
	if *blk {
		mountBlk()
	}

	for _, path := range verify {
		f, err := openAndVerify(path, key)
		if err != nil {
			log.Fatalf("openAndVerify(%s): %v", path, err)
		}
		f.Close()
	}
	log.Printf("Integrity check OK")

	// Try kexec_file_load first
	kernel, err := os.Open(kernelPath)
	if err != nil {
		log.Fatalf("Open(%s): %v", kernelPath, err)
	}
	err = kexec.FileLoad(kernel, nil, "")
	if err != nil {
		log.Fatalf("KexecFileLoad: %v", err)
	}
	kernel.Close()
	log.Print("Looks like kexec_file_load didn't work, let's try kexec_load")

	// If kexec_file_load fails try kexec_load second
	image := &boot.LinuxImage{
		Kernel: uio.NewLazyFile(kernelPath),
	}
	err = image.Load(true)
	if err != nil {
		log.Fatalf("Load(%s): %v", kernelPath, err)
	}
	err = kexec.Reboot()
	if err != nil {
		log.Fatalf("Reboot: %v", err)
	}
}

// createBasicHirarchy creates some basic directories and mounts if they don't exist yet
func createBasicHirarchy() {
	// Create base directories
	dirs := []string{"/mnt", "/ro", "/tmp", "/proc", "/sys", "/dev"}
	for _, dir := range dirs {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			log.Fatalf("Mkdir(%s): %v", dir, err)
		}
	}

	// Mount base directories
	mnts := []string{"sysfs", "proc", "devtmpfs", "tmpfs"}
	for _, mnt := range mnts {
		mount(mnt)
	}

	// Set up remaining parts for /dev
	err := os.MkdirAll("/dev/pts", 0755)
	if err != nil {
		log.Fatalf("Mkdir(/dev/pts): %v", err)
	}
	mount("devpts")
	// err = os.Symlink("/dev/pts/ptmx", "/dev/ptmx")
	// if err != nil {
	// 	log.Fatalf("Symlink(/dev/ptmx): %v", err)
	// }

}

// mountMtd mounts u-bmc on MTD flash
func mountMtd() {
	err := unix.Mount("ubi0:root", "/ro", "ubifs", unix.MS_RDONLY, "")
	if err != nil {
		log.Fatalf("Mount(ubi0:root): %v", err)
	}
}

// mountBlk mounts u-bmc on a block device
func mountBlk() {
	var offset int64 = 1072 // This is the offset at which erofs stores the UUID
	var data = make([]byte, 16)
	var uuid = []byte{0x26, 0xab, 0x04, 0x01, 0x3f, 0x49, 0x4f, 0xc2, 0xa1, 0x72, 0xc8, 0xaa, 0x02, 0xac, 0xea, 0xf3}

	devs, _ := filepath.Glob("/sys/class/block/*")
	for _, dev := range devs {
		dev = "/dev/" + filepath.Base(dev)
		bd, err := os.Open(dev)
		if err != nil {
			log.Fatalf("Open(%s): %v", dev, err)
		}
		_, err = bd.ReadAt(data, offset)
		bd.Close()
		if err == nil && bytes.Equal(data, uuid) {
			err = unix.Mount(dev, "/ro", "erofs", unix.MS_RDONLY, "")
			if err != nil {
				log.Fatalf("Mount(%s): %v", dev, err)
			}
		}
	}
}

// mountOverlay mounts the overlayfs on top of the ro root
func mountOverlay() {
	tmpdirs := []string{"/tmp/upper", "/tmp/work"}
	for _, dir := range tmpdirs {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			log.Fatalf("Mkdir(%s): %v", dir, err)
		}
	}
	mount("overlayfs")
}

// Abstraction for unix.Mount
func mount(fs string) {
	var err error
	switch fs {
	case "proc":
		err = unix.Mount(fs, "/proc", fs, 0, "")
	case "sysfs":
		err = unix.Mount(fs, "/sys", fs, 0, "")
	case "devtmpfs":
		err = unix.Mount(fs, "/dev", fs, 0, "")
	case "devpts":
		err = unix.Mount(fs, "/dev/pts", fs, 0, "newinstance,ptmxmode=666,gid=5,mode=620")
	case "tmpfs":
		err = unix.Mount(fs, "/tmp", fs, 0, "")
	case "overlayfs":
		err = unix.Mount(fs, "/mnt", "overlay", 0, "lowerdir=/ro,upperdir=/tmp/upper,workdir=/tmp/work")
	}
	if err != nil {
		log.Fatalf("Mount(%s): %v", fs, err)
	}
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
