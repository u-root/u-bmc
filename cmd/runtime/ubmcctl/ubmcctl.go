// Copyright 2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"

	"github.com/u-root/u-bmc/pkg/service/logger"
	"google.golang.org/grpc"
)

var (
	log  = logger.LogContainer.GetSimpleLogger()
	host = flag.String("host", "localhost", "Which u-bmc host to connect to")
	ui   = flag.Bool("ui", false, "Run ubmcctl in UI mode [WIP]")
)

func main() {
	flag.Parse()

	if *ui {
		log.Warn("This mode is not implemented yet")
		return
	}

	var target string
	if *host == "localhost" {
		// Connect to localhost:80 unauthenticated and unecrypted
		// This is used to troubleshoot on the BMC. If you have shell access
		// on the BMC you're already authorized to execute RPCs.
		target = "127.0.0.1:80"
	} else {
		target = *host
	}
	conn := newConnection(target)
	defer conn.Close()
	client := newClient(conn)
	if len(flag.Args()) == 0 {
		usage(conn)
	} else {
		callRPC(client, flag.Args())
	}
}

func usage(conn *grpc.ClientConn) {
	services := listServices(conn)
	fmt.Println("Found following services:")
	for _, service := range services {
		fmt.Println(service)
	}
}
