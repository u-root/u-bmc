// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// Loads an RSA key from boot/keys/u-bmc.key and signs the stdin
// using GPG and emits the result on stdout.
//
// If $(SRC)/boot/keys/u-bmc.pub is not present, it is created.
//
// TODO(bluecmd): Since u-boot (at least 2016 version) does not support
// eliptic curve RSA is being used.

package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
	"golang.org/x/crypto/openpgp/s2k"
)

var (
	privateKeyPath  = "./keys/u-bmc.key"
	publicKeyPath   = "./keys/u-bmc.pub"
	keyLifetimeSecs = uint32(86400 * 365 * 100)
)

func main() {
	// Expand key paths from directory where binary is to allow running
	// from e.g. integration tests
	privateKeyPath = filepath.Join(filepath.Dir(os.Args[0]), privateKeyPath)
	publicKeyPath = filepath.Join(filepath.Dir(os.Args[0]), publicKeyPath)

	flag.Parse()
	config := packet.Config{
		DefaultHash:   crypto.SHA256,
		DefaultCipher: packet.CipherAES256,
		Time:          time.Now,
	}

	pkData, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		log.Fatalf("ioutil.ReadFile(%s): %v", privateKeyPath, err)
	}
	pkPem, _ := pem.Decode(pkData)
	privKey, err := x509.ParsePKCS1PrivateKey(pkPem.Bytes)
	if err != nil {
		log.Fatalf("x509.ParsePKCS1PrivateKey: %v", err)
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
		log.Fatalf("Could not resolve %v", config.DefaultHash)
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

	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		o, err := os.OpenFile(publicKeyPath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatalf("os.OpenFile(%s): %v", publicKeyPath, err)
		}
		defer o.Close()
		pubKey.Serialize(o)
	}

	err = openpgp.DetachSign(os.Stdout, &e, os.Stdin, nil)
	if err != nil {
		log.Fatalf("openpgp.DetachSign: %v", err)
	}
}
