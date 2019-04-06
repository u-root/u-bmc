// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"

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

	fmt.Printf(`
██╗   ██╗      ██████╗ ███╗   ███╗ ██████╗
██║   ██║      ██╔══██╗████╗ ████║██╔════╝
██║   ██║█████╗██████╔╝██╔████╔██║██║
██║   ██║╚════╝██╔══██╗██║╚██╔╝██║██║
╚██████╔╝      ██████╔╝██║ ╚═╝ ██║╚██████╗
 ╚═════╝       ╚═════╝ ╚═╝     ╚═╝ ╚═════╝

`)

	envv := []string{"TZ=UTC", "HOME=/root", "USER=root", "PATH=/bbin:/bin"}
	err = unix.Exec(*shell, []string{*shell}, envv)
	log.Fatalf("Failed to exec: %v", err)
}
