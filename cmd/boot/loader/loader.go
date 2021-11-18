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
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"aead.dev/minisign"
	"github.com/u-root/u-bmc/pkg/logger"
	"github.com/u-root/u-root/pkg/boot"
	"github.com/u-root/u-root/pkg/boot/kexec"
	"github.com/u-root/u-root/pkg/kmodule"
	uroot "github.com/u-root/u-root/pkg/mount"
	"github.com/u-root/u-root/pkg/uio"
	"golang.org/x/sys/unix"
)

const (
	pubKeyPath = "/u-bmc.pub"
	kernelPath = "/ro/boot/zImage"
	dtbPath    = "/ro/boot/platform.dtb"
	initPath   = "/ro/bin/init"
)

var (
	kload    = flag.Bool("kexec", false, "Mount rootfs and call kexec")
	swroot   = flag.Bool("switch", false, "Mount rootfs and call switch_root")
	mtd      = flag.Bool("mtd", false, "Mount and load u-bmc from MTD flash")
	blk      = flag.Bool("blk", false, "Mount and load u-bmc from block device")
	ast      = flag.Bool("ast", false, "ASPEED ast specific option")
	toVerify = []string{initPath, kernelPath, dtbPath}
	lc       = logger.LogContainer
	log      = lc.GetLogger()
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
		loadModule("/bootlock.ko")
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

	check(uroot.SwitchRoot("/mnt", "/bin/init"), "Failed to execute SwitchRoot")
}

// loadAndExec validates boot files and runs them via kexec
func loadAndExec() {
	createBasicHirarchy()
	if *mtd {
		mountMtd()
	}
	if *blk {
		mountBlk()
	}

	for _, path := range toVerify {
		verify(path)
	}
	log.Info("Integrity check OK")

	// Try kexec_file_load first
	kernel, err := os.Open(kernelPath)
	check(err, fmt.Sprintf("Failed to open %s", kernelPath))
	check(kexec.FileLoad(kernel, nil, ""), "Failed to call KexecFileLoad")
	kernel.Close()
	log.Info("Looks like kexec_file_load didn't work, let's try kexec_load")

	// If kexec_file_load fails try kexec_load second
	image := &boot.LinuxImage{
		Kernel: uio.NewLazyFile(kernelPath),
	}
	check(image.Load(true), "Failed to load kernel")
	check(kexec.Reboot(), "Failed to call kexec")
}

// createBasicHirarchy creates some basic directories and mounts if they don't exist yet
func createBasicHirarchy() {
	// Create base directories
	dirs := []string{"/mnt", "/ro", "/tmp", "/proc", "/sys", "/dev"}
	for _, dir := range dirs {
		check(os.MkdirAll(dir, 0755), fmt.Sprintf("Failed to create %s", dir))
	}

	// Mount base directories
	mnts := []string{"sysfs", "proc", "devtmpfs", "tmpfs"}
	for _, mnt := range mnts {
		mount(mnt)
	}

	// Set up remaining parts for /dev
	check(os.MkdirAll("/dev/pts", 0755), "Failed to create /dev/pts")
	mount("devpts")
	//check(os.Symlink("/dev/pts/ptmx", "/dev/ptmx"), "Failed to symlink /dev/ptmx")
}

// mountMtd mounts u-bmc on MTD flash
func mountMtd() {
	check(unix.Mount("ubi0:root", "/ro", "ubifs", unix.MS_RDONLY, ""), "Failed to mount ubi0:root")
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
		check(err, fmt.Sprintf("Failed to open %s", dev))
		_, err = bd.ReadAt(data, offset)
		bd.Close()
		if err == nil && bytes.Equal(data, uuid) {
			check(unix.Mount(dev, "/ro", "erofs", unix.MS_RDONLY, ""), fmt.Sprintf("Failed to mount %s", dev))
		}
	}
}

// mountOverlay mounts the overlayfs on top of the ro root
func mountOverlay() {
	tmpdirs := []string{"/tmp/upper", "/tmp/work"}
	for _, dir := range tmpdirs {
		check(os.MkdirAll(dir, 0755), fmt.Sprintf("Failed to create %s", dir))
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
	check(err, fmt.Sprintf("Failed mounting %s", fs))
}

func verify(filePath string) {
	f, err := os.ReadFile(filePath)
	check(err, fmt.Sprintf("Failed to open %s", filePath))

	sig, err := os.ReadFile(filePath + ".sig")
	check(err, fmt.Sprintf("Failed to open %s", filePath+".sig"))

	key, err := minisign.PublicKeyFromFile(pubKeyPath)
	check(err, fmt.Sprintf("Failed to read public key %s", pubKeyPath))

	if !minisign.Verify(key, f, sig) {
		log.Fatal(fmt.Sprintf("Signature for %s does not match, bailing out!", filePath))
	}
}

func loadModule(fp string) {
	f, err := os.Open(fp)
	check(err, fmt.Sprintf("Failed to open kmod %s", fp))

	defer f.Close()
	check(kmodule.FileInit(f, "", 0), "Failed loading kernel")
}

func check(err error, msg string) {
	if err != nil {
		log.Fatal(msg, lc.String("err", err.Error()))
	}
}
