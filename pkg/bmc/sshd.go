// Copyright 2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type SshServer struct {
	conf *ssh.ServerConfig
}

func RunSshServer(authorizedKeys []string) error {
	err := createFile("/config/authorized_keys", 0600, []byte(strings.Join(authorizedKeys, "\n")))
	if err != nil {
		return fmt.Errorf("failed to create authorized_keys: %v", err)
	}

	authorizedKeysBytes, err := os.ReadFile("/config/authorized_keys")
	if err != nil {
		return fmt.Errorf("failed to load authorized_keys, err: %v", err)
	}

	authorizedKeysMap := map[string]bool{}
	for len(authorizedKeysBytes) > 0 {
		pubKey, _, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
		if err != nil {
			return fmt.Errorf(err.Error())
		}

		authorizedKeysMap[string(pubKey.Marshal())] = true
		authorizedKeysBytes = rest
	}

	// An SSH server is represented by a ServerConfig, which holds
	// certificate details and handles authentication of ServerConns.
	config := &ssh.ServerConfig{
		// Remove to disable password auth.
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// Should use constant-time compare (or better, salt+hash) in
			// a production setting.
			if c.User() == "root" && string(pass) == "test" {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", c.User())
		},

		// Remove to disable public key auth.
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			if authorizedKeysMap[string(pubKey.Marshal())] {
				return &ssh.Permissions{
					// Record the public key used for authentication.
					Extensions: map[string]string{
						"pubkey-fp": ssh.FingerprintSHA256(pubKey),
					},
				}, nil
			}
			return nil, fmt.Errorf("unknown public key for %q", c.User())
		},
	}

	privateBytes, err := os.ReadFile("/config/ssh_host_ed25519_key")
	if err != nil {
		return fmt.Errorf("failed to load private key: %v", err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %v", err)
	}

	config.AddHostKey(private)

	go startSshServer(config)

	return nil
}

func SshKeyGen(privKeyPath string, pubKeyPath string) error {
	rand, err := os.Open("/dev/random")
	if err != nil {
		return fmt.Errorf("opening /dev/random failed: %v", err)
	}
	defer rand.Close()
	pubKey, privKey, err := ed25519.GenerateKey(rand)
	if err != nil {
		return fmt.Errorf("generating keys failed: %v", err)
	}
	if privKeyPath != "" {
		asn1, err := x509.MarshalPKCS8PrivateKey(privKey)
		if err != nil {
			return fmt.Errorf("mashalling private key failed: %v", err)
		}
		err = os.WriteFile(privKeyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: asn1}), 0644)
		if err != nil {
			return fmt.Errorf("failed writing private key: %v", err)
		}
	}
	if pubKeyPath != "" {
		err = os.WriteFile(pubKeyPath, pubKey, 0644)
		if err != nil {
			return fmt.Errorf("failed writing public key: %v", err)
		}
	}
	return nil
}

func startSshServer(config *ssh.ServerConfig) error {
	listener, err := net.Listen("tcp", ":22")
	if err != nil {
		return fmt.Errorf("failed to listen for connection: %v", err)
	}

	tcpConn, _ := listener.Accept()
	if err != nil {
		return fmt.Errorf("failed to accept incoming connection: %v", err)
	}
	conn, chans, reqs, _ := ssh.NewServerConn(tcpConn, config)
	if err != nil {
		return fmt.Errorf("failed to handshake: %v", err)
	}
	if conn == nil {
		return fmt.Errorf("no connection established")
	}

	go handleChannels(chans)
	go ssh.DiscardRequests(reqs)

	return nil
}

func handleChannels(chans <-chan ssh.NewChannel) {
	for newChannel := range chans {
		go handleChannel(newChannel)
	}
}

func handleChannel(newChannel ssh.NewChannel) {
	t := newChannel.ChannelType()
	if t != "session" {
		newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
		return
	}

	channel, requests, err := newChannel.Accept()
	if err != nil {
		log.Warnf("Could not accept SSH channel: %v", err)
	}

	go handleRequests(channel, requests)
}

func handleRequests(channel ssh.Channel, requests <-chan *ssh.Request) {
	for req := range requests {
		switch req.Type {
		case "shell":
			err := attachShell(channel)
			req.Reply(err == nil, nil)
		default:
			log.Debugf("unhandled SSH request: %s (reply: %v, data: %x)", req.Type, req.WantReply, req.Payload)
		}
	}
}

func attachShell(channel ssh.Channel) error {

	t := term.NewTerminal(channel, "=> ")
	for {
		line, err := t.ReadLine()
		if err != nil {
			break
		}
		fmt.Println(line)
	}

	return nil
}
