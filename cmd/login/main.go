// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sys/unix"
)

var (
	shell = flag.String("shell", "/bbin/elvish", "Shell to login to")
)

func main() {
	flag.Parse()

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("")
	fmt.Println("Press enter to activate terminal")
	_, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Unable to read from terminal: %v", err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "<unknown>"
	}

	fmt.Printf(`
██╗   ██╗      ██████╗ ███╗   ███╗ ██████╗
██║   ██║      ██╔══██╗████╗ ████║██╔════╝
██║   ██║█████╗██████╔╝██╔████╔██║██║
██║   ██║╚════╝██╔══██╗██║╚██╔╝██║██║
╚██████╔╝      ██████╔╝██║ ╚═╝ ██║╚██████╗
 ╚═════╝       ╚═════╝ ╚═╝     ╚═╝ ╚═════╝

Note: This is TODO, password is 'rosebud'
%s login: `, hostname)

	pw, _ := terminal.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println("")
	if string(pw) != "rosebud" {
		fmt.Println("Password incorrect")
		return
	}
	fmt.Println("")

	if err := ioutil.WriteFile("/proc/self/attr/exec", []byte("exec shell\000"), 0600); err != nil {
		log.Fatalf("Failed to set profile: %v", err)
	}

	envv := []string{"TZ=UTC", "HOME=/root", "USER=root", "PATH=/bbin:/bin"}
	err = unix.Exec(*shell, []string{*shell}, envv)
	log.Fatalf("Failed to exec: %v", err)
}
