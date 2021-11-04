// Copyright 2019 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/u-root/u-bmc/pkg/logger"
	"src.elv.sh/pkg/buildinfo"
	"src.elv.sh/pkg/prog"
	"src.elv.sh/pkg/shell"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	log := logger.LogContainer.GetSimpleLogger()

	log.Info("\033[32mPress ENTER to activate the terminal\033[0m")
	_, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Unable to read from terminal: %v", err)
	}

	fmt.Print(` ___ _  _ ___ _    _    
/ __| || | __| |  | |   
\__ \ __ | _|| |__| |__ 
|___/_||_|___|____|____|						

`)
	// TODO(MDr164): Make use of daemon mode
	os.Exit(prog.Run([3]*os.File{os.Stdin, os.Stdout, os.Stderr}, os.Args,
		buildinfo.Program, daemonStub{}, shell.Program{}))
}

type daemonStub struct{}

func (daemonStub) ShouldRun(f *prog.Flags) bool {
	return f.Daemon
}

func (daemonStub) Run(fds [3]*os.File, f *prog.Flags, args []string) error {
	return fmt.Errorf("daemon mode not supported in this build")
}
