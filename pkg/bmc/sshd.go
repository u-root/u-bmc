// Copyright 2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/creack/pty"
	"golang.org/x/crypto/ssh"
)

// SSHServer represents a SSHD server with a specific configuration
type SSHServer struct {
	conf *ssh.ServerConfig
}

// LaunchSSHServer takes a string array of public keys and sets up everything
// for an SSHD server that only allows pubkey authentication. The resulting
// subprocess will listen on port 22 as usual.
func (s SSHServer) LaunchSSHServer(authorizedKeys []string) error {
	// Create authorized_keys file with provided keys
	err := createFile("/config/authorized_keys", 0600, []byte(strings.Join(authorizedKeys, "\n")))
	if err != nil {
		return fmt.Errorf("failed to create authorized_keys: %v", err)
	}
	// Read out keys and keep in memory
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
	// Create SSH configuration that only allows pubkey auth
	s.conf = &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			if authorizedKeysMap[string(pubKey.Marshal())] {
				return &ssh.Permissions{
					Extensions: map[string]string{
						"pubkey-fp": ssh.FingerprintSHA256(pubKey),
					},
				}, nil
			}
			return nil, fmt.Errorf("unknown public key for %q", c.User())
		},
	}
	// Read and load private host key
	privateBytes, err := os.ReadFile("/config/ssh_host_ed25519_key")
	if err != nil {
		return fmt.Errorf("failed to load private key: %v", err)
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %v", err)
	}
	s.conf.AddHostKey(private)
	// Launch actual server in a new goroutine
	go s.startSSHServer()

	return nil
}

// SSHKeyGen generates an ED25519 SSH key pair
func (s SSHServer) SSHKeyGen(privKeyPath string, pubKeyPath string) error {
	// Generate key pair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generating keys failed: %v", err)
	}
	// Write out private key if path is given
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
	// Write out public key if path is given
	if pubKeyPath != "" {
		err = os.WriteFile(pubKeyPath, pubKey, 0644)
		if err != nil {
			return fmt.Errorf("failed writing public key: %v", err)
		}
	}
	return nil
}

func (s SSHServer) startSSHServer() error {
	// Listen on default SSH port
	listener, err := net.Listen("tcp", ":22")
	if err != nil {
		return fmt.Errorf("failed to listen for connection: %v", err)
	}
	// Accept incoming TCP connection and do handshakes
	tcpConn, _ := listener.Accept()
	if err != nil {
		return fmt.Errorf("failed to accept incoming connection: %v", err)
	}
	conn, chans, reqs, _ := ssh.NewServerConn(tcpConn, s.conf)
	if err != nil {
		return fmt.Errorf("failed to handshake: %v", err)
	}
	if conn == nil {
		return fmt.Errorf("no connection established")
	}
	// Handle connections in a new goroutine
	go handleChannels(chans)
	// Discard out-of-band requests in a new goroutine
	go ssh.DiscardRequests(reqs)

	return nil
}

func handleChannels(chans <-chan ssh.NewChannel) {
	// Handle each connection in its own goroutine
	for newChannel := range chans {
		go handleChannel(newChannel)
	}
}

func handleChannel(newChannel ssh.NewChannel) {
	// We only handle session type connections
	t := newChannel.ChannelType()
	if t != "session" {
		newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
		return
	}
	channel, requests, err := newChannel.Accept()
	if err != nil {
		log.Warnf("Could not accept SSH channel: %v", err)
	}
	// Handle requests in its own goroutine
	go handleRequests(channel, requests)
}

func handleRequests(channel ssh.Channel, requests <-chan *ssh.Request) {
	for req := range requests {
		// For now we only handle shell type requests
		switch req.Type {
		case "shell":
			err := attachShell(channel)
			req.Reply(err == nil, nil)
		default:
			log.Debugf("unhandled SSH request: %s (reply: %v, data: %x)", req.Type, req.WantReply, req.Payload)
		}
	}
}

//TODO(MDr164): Don't use exec.Cmd here but rather implement a proper
// in-process shell like we do in the login cmd
func attachShell(channel ssh.Channel) error {
	sh := exec.Command("elvish")
	close := func() {
		channel.Close()
		if sh.Process != nil {
			sh.Process.Wait()
		}
	}
	shf, err := pty.Start(sh)
	if err != nil {
		close()
		return err
	}
	go func() {
		io.Copy(channel, shf)
	}()
	go func() {
		io.Copy(shf, channel)
	}()

	return nil
}
