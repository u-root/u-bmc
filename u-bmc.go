// Copyright 2015-2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/u-root/u-root/pkg/cpio"
	"github.com/u-root/u-root/pkg/golang"
	"github.com/u-root/u-root/pkg/uroot"
	"github.com/u-root/u-root/pkg/uroot/builder"
	"github.com/u-root/u-root/pkg/uroot/initramfs"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
	"golang.org/x/crypto/openpgp/s2k"
)

var (
	privateKeyPath  = "boot/keys/u-bmc.key"
	keyLifetimeSecs = uint32(86400 * 365 * 100) // 100 years

	outputPath = flag.String("o", "", "Path to output initramfs file.")
	build      = flag.String("build", "", "Build is either 'u-bmc' or 'loader'.")
	platform   = flag.String("p", "", "Platform to target.")
	packages   = []string{
		"github.com/u-root/u-bmc/cmd/fan",
		"github.com/u-root/u-bmc/cmd/i2cwatcher",
		"github.com/u-root/u-bmc/cmd/socreset",
		"github.com/u-root/u-bmc/cmd/ubmcctl",
		// Based on core in u-root
		"github.com/u-root/u-root/cmds/ansi",
		"github.com/u-root/u-root/cmds/boot",
		"github.com/u-root/u-root/cmds/cat",
		"github.com/u-root/u-root/cmds/cbmem",
		"github.com/u-root/u-root/cmds/chmod",
		"github.com/u-root/u-root/cmds/chroot",
		"github.com/u-root/u-root/cmds/cmp",
		"github.com/u-root/u-root/cmds/console",
		"github.com/u-root/u-root/cmds/cp",
		"github.com/u-root/u-root/cmds/cpio",
		"github.com/u-root/u-root/cmds/date",
		"github.com/u-root/u-root/cmds/dd",
		"github.com/u-root/u-root/cmds/df",
		"github.com/u-root/u-root/cmds/dhclient",
		"github.com/u-root/u-root/cmds/dirname",
		"github.com/u-root/u-root/cmds/dmesg",
		"github.com/u-root/u-root/cmds/echo",
		"github.com/u-root/u-root/cmds/elvish",
		"github.com/u-root/u-root/cmds/false",
		"github.com/u-root/u-root/cmds/field",
		"github.com/u-root/u-root/cmds/find",
		"github.com/u-root/u-root/cmds/free",
		"github.com/u-root/u-root/cmds/freq",
		"github.com/u-root/u-root/cmds/gpgv",
		"github.com/u-root/u-root/cmds/gpt",
		"github.com/u-root/u-root/cmds/grep",
		"github.com/u-root/u-root/cmds/gzip",
		"github.com/u-root/u-root/cmds/hexdump",
		"github.com/u-root/u-root/cmds/hostname",
		"github.com/u-root/u-root/cmds/id",
		"github.com/u-root/u-root/cmds/init",
		"github.com/u-root/u-root/cmds/insmod",
		"github.com/u-root/u-root/cmds/installcommand",
		"github.com/u-root/u-root/cmds/io",
		"github.com/u-root/u-root/cmds/ip",
		"github.com/u-root/u-root/cmds/kexec",
		"github.com/u-root/u-root/cmds/kill",
		"github.com/u-root/u-root/cmds/lddfiles",
		"github.com/u-root/u-root/cmds/ln",
		"github.com/u-root/u-root/cmds/losetup",
		"github.com/u-root/u-root/cmds/ls",
		"github.com/u-root/u-root/cmds/lsmod",
		"github.com/u-root/u-root/cmds/mkdir",
		"github.com/u-root/u-root/cmds/mkfifo",
		"github.com/u-root/u-root/cmds/mknod",
		"github.com/u-root/u-root/cmds/modprobe",
		"github.com/u-root/u-root/cmds/mount",
		"github.com/u-root/u-root/cmds/msr",
		"github.com/u-root/u-root/cmds/mv",
		"github.com/u-root/u-root/cmds/netcat",
		"github.com/u-root/u-root/cmds/ntpdate",
		"github.com/u-root/u-root/cmds/pci",
		"github.com/u-root/u-root/cmds/ping",
		"github.com/u-root/u-root/cmds/printenv",
		"github.com/u-root/u-root/cmds/ps",
		"github.com/u-root/u-root/cmds/pwd",
		"github.com/u-root/u-root/cmds/pxeboot",
		"github.com/u-root/u-root/cmds/readlink",
		"github.com/u-root/u-root/cmds/rm",
		"github.com/u-root/u-root/cmds/rmmod",
		"github.com/u-root/u-root/cmds/rsdp",
		"github.com/u-root/u-root/cmds/scp",
		"github.com/u-root/u-root/cmds/seq",
		"github.com/u-root/u-root/cmds/shutdown",
		"github.com/u-root/u-root/cmds/sleep",
		"github.com/u-root/u-root/cmds/sort",
		"github.com/u-root/u-root/cmds/sshd",
		"github.com/u-root/u-root/cmds/stty",
		"github.com/u-root/u-root/cmds/switch_root",
		"github.com/u-root/u-root/cmds/sync",
		"github.com/u-root/u-root/cmds/tail",
		"github.com/u-root/u-root/cmds/tee",
		"github.com/u-root/u-root/cmds/true",
		"github.com/u-root/u-root/cmds/truncate",
		"github.com/u-root/u-root/cmds/umount",
		"github.com/u-root/u-root/cmds/uname",
		"github.com/u-root/u-root/cmds/uniq",
		"github.com/u-root/u-root/cmds/unshare",
		"github.com/u-root/u-root/cmds/validate",
		"github.com/u-root/u-root/cmds/vboot",
		"github.com/u-root/u-root/cmds/wc",
		"github.com/u-root/u-root/cmds/wget",
		"github.com/u-root/u-root/cmds/which",
	}
)

type signedBuilder struct {
	e  *openpgp.Entity
	bb *builder.BBBuilder
}

func (b *signedBuilder) DefaultBinaryDir() string {
	return b.bb.DefaultBinaryDir()
}

func (b signedBuilder) Build(af *initramfs.Files, opts builder.Opts) error {
	if err := b.bb.Build(af, opts); err != nil {
		return err
	}

	bbpath := filepath.Join(opts.TempDir, "bb")
	bbsigpath := bbpath + ".sig"
	pkpath := filepath.Join(opts.TempDir, "etc", "u-bmc.pub")
	if err := os.MkdirAll(filepath.Dir(pkpath), 0755); err != nil {
		return err
	}

	if err := b.writeCryptoFiles(bbpath, bbsigpath, pkpath); err != nil {
		return err
	}

	if err := af.AddFile(bbsigpath, filepath.Join(opts.BinaryDir, "bb.sig")); err != nil {
		return err
	}
	if err := af.AddRecord(cpio.Symlink("init.sig", filepath.Join(opts.BinaryDir, "bb.sig"))); err != nil {
		return err
	}
	_ = af.AddRecord(cpio.Directory("etc", 0755))
	if err := af.AddFile(pkpath, filepath.Join("etc", "u-bmc.pub")); err != nil {
		return err
	}
	return nil
}

func (b signedBuilder) writeCryptoFiles(bb string, bbsig string, pk string) error {
	bbf, err := os.Open(bb)
	if err != nil {
		return err
	}
	defer bbf.Close()

	bbsf, err := os.OpenFile(bbsig, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer bbsf.Close()

	err = openpgp.DetachSign(bbsf, b.e, bbf, nil)
	if err != nil {
		return err
	}

	pkf, err := os.OpenFile(pk, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer pkf.Close()
	return b.e.PrimaryKey.Serialize(pkf)
}

func main() {
	flag.Parse()

	e, err := BuildSigner()
	if err != nil {
		log.Fatal(err)
	}

	env := golang.Default()
	if env.CgoEnabled {
		// TODO(bluecmd): Might need CGO for pcap if that should be included
		// Given that we already depend on a gcc being available for u-boot and
		// the linux kernel, this might be fine. Especially if we need to do the
		// yocto route down the line.
		log.Printf("Disabling CGO for u-bmc...")
		env.CgoEnabled = false
	}
	log.Printf("Build environment: %s", env)
	if env.GOOS != "linux" {
		log.Printf("GOOS is not linux. Did you mean to set GOOS=linux?")
	}

	// TODO(bluecmd): Read from platform definitions
	env.GOARCH = "arm"
	// TODO(bluecmd): Fix u-root builder so that GOARM can be specified
	// env.GOARM = 5

	if *build == "u-bmc" {
		if err := BuildUBMC(e, &env); err != nil {
			log.Fatal(err)
		}
		log.Printf("Successfully wrote u-bmc initramfs.")
	} else if *build == "loader" {
		if err := BuildLoader(e, &env); err != nil {
			log.Fatal(err)
		}
		log.Printf("Successfully wrote loader initramfs.")
	} else {
		log.Fatal("Unknown build mode")
	}
}

func BuildLoader(e *openpgp.Entity, env *golang.Environ) error {
	builder := &builder.BinaryBuilder{}
	archiver, err := initramfs.GetArchiver("cpio")
	if err != nil {
		return err
	}

	tempDir, err := ioutil.TempDir("", "u-bmc")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	w, err := archiver.OpenWriter(*outputPath, env.GOOS, env.GOARCH)
	if err != nil {
		return err
	}

	pkgs := []string{"github.com/u-root/u-bmc/cmd/loader"}
	baseFile := uroot.DefaultRamfs.Reader()

	opts := uroot.Opts{
		Env: *env,
		Commands: []uroot.Commands{
			{
				Builder:  builder,
				Packages: pkgs,
			},
		},
		BaseArchive: baseFile,
		TempDir:     tempDir,
		OutputFile:  w,
		InitCmd:     "loader",
	}
	logger := log.New(os.Stderr, "", log.LstdFlags)
	return uroot.CreateInitramfs(logger, opts)
}

func BuildUBMC(e *openpgp.Entity, env *golang.Environ) error {
	builder := &signedBuilder{e, &builder.BBBuilder{}}
	archiver, err := initramfs.GetArchiver("cpio")
	if err != nil {
		return err
	}

	tempDir, err := ioutil.TempDir("", "u-bmc")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	w, err := archiver.OpenWriter(*outputPath, env.GOOS, env.GOARCH)
	if err != nil {
		return err
	}

	pkgs := append(packages, fmt.Sprintf("github.com/u-root/u-bmc/platform/%s/cmd/*", *platform))

	baseFile := uroot.DefaultRamfs.Reader()

	opts := uroot.Opts{
		Env: *env,
		Commands: []uroot.Commands{
			{
				Builder:  builder,
				Packages: pkgs,
			},
		},
		BaseArchive:  baseFile,
		TempDir:      tempDir,
		OutputFile:   w,
		InitCmd:      "init",
		DefaultShell: "elvish",
	}
	logger := log.New(os.Stderr, "", log.LstdFlags)
	return uroot.CreateInitramfs(logger, opts)
}

func BuildSigner() (*openpgp.Entity, error) {
	// Expand key paths from directory where binary is to allow running
	// from e.g. integration tests
	privateKeyPath = filepath.Join(filepath.Dir(os.Args[0]), privateKeyPath)

	config := packet.Config{
		DefaultHash:   crypto.SHA256,
		DefaultCipher: packet.CipherAES256,
		Time:          time.Now,
	}

	pkData, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("ioutil.ReadFile(%s): %v", privateKeyPath, err)
	}
	pkPem, _ := pem.Decode(pkData)
	privKey, err := x509.ParsePKCS1PrivateKey(pkPem.Bytes)
	if err != nil {
		return nil, fmt.Errorf("x509.ParsePKCS1PrivateKey: %v", err)
	}

	ct := config.Time()
	uid := packet.NewUserId("u-bmc builder", "", "")
	pubKey := packet.NewRSAPublicKey(ct, privKey.Public().(*rsa.PublicKey))
	e := openpgp.Entity{
		PrimaryKey: pubKey,
		PrivateKey: packet.NewRSAPrivateKey(ct, privKey),
		Identities: make(map[string]*openpgp.Identity),
	}
	isPrimaryId := true
	e.Identities[uid.Id] = &openpgp.Identity{
		Name:   uid.Name,
		UserId: uid,
		SelfSignature: &packet.Signature{
			CreationTime: ct,
			FlagCertify:  true,
			FlagSign:     true,
			FlagsValid:   true,
			Hash:         config.Hash(),
			IsPrimaryId:  &isPrimaryId,
			IssuerKeyId:  &e.PrimaryKey.KeyId,
			PubKeyAlgo:   packet.PubKeyAlgoRSA,
			SigType:      packet.SigTypePositiveCert,
		},
	}
	hid, ok := s2k.HashToHashId(config.DefaultHash)
	if !ok {
		return nil, fmt.Errorf("Could not resolve %v", config.DefaultHash)
	}
	e.Subkeys = make([]openpgp.Subkey, 1)
	e.Subkeys[0] = openpgp.Subkey{
		PublicKey:  pubKey,
		PrivateKey: packet.NewRSAPrivateKey(ct, privKey),
		Sig: &packet.Signature{
			CreationTime:              ct,
			FlagEncryptCommunications: true,
			FlagEncryptStorage:        true,
			FlagsValid:                true,
			Hash:                      config.Hash(),
			IssuerKeyId:               &e.PrimaryKey.KeyId,
			KeyLifetimeSecs:           &keyLifetimeSecs,
			PreferredHash:             []uint8{hid},
			PubKeyAlgo:                packet.PubKeyAlgoRSA,
			SigType:                   packet.SigTypeSubkeyBinding,
		},
	}
	return &e, nil
}
