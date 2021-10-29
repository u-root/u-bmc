// Copyright 2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// Tool to create key pairs and sign files with them using
// minisign. It uses ED25519 and BLAKE2b under the hood.

package main

import (
	"flag"
	"os"
	"path/filepath"

	"aead.dev/minisign"
	"github.com/u-root/u-bmc/pkg/logger"
)

var (
	gen            = flag.Bool("gen", false, "Generates a key pair to sign files with")
	sign           = flag.String("sign", "", "File to sign")
	privateKeyPath = "keys/u-bmc.key"
	publicKeyPath  = "keys/u-bmc.pub"
)

func main() {
	flag.Parse()

	if *gen {
		generateKeypair()
		return
	}

	if *sign != "" {
		path, err := filepath.Abs(*sign)
		check(err, "Failed to evaluate file path:")
		signFile(path)
	}
}

func generateKeypair() {
	_, err := os.Stat(privateKeyPath)
	if os.IsExist(err) {
		check(err, "Private key already exists:")
	}
	publicKey, privateKey, err := minisign.GenerateKey(nil)
	check(err, "Failed to generate key pair:")

	pk, err := minisign.EncryptKey("ubmc", privateKey)
	check(err, "Failed to transform key:")

	check(os.WriteFile(privateKeyPath, pk, 0600), "Failed to write private key:")
	check(os.WriteFile(publicKeyPath, []byte(publicKey.String()), 0644), "Failed to write public key:")
}

func signFile(path string) {
	key, err := minisign.PrivateKeyFromFile("ubmc", "./boot/"+privateKeyPath)
	check(err, "Failed reading private key:")

	f, err := os.ReadFile(path)
	check(err, "Failed to read file to sign:")

	_, err = os.Stdout.Write(minisign.Sign(key, f))
	check(err, "Failed to create signature:")
}

func check(err error, msg string) {
	lc := logger.LogContainer
	if err != nil {
		lc.GetLogger().Error(msg, lc.String("err", err.Error()))
	}
}
