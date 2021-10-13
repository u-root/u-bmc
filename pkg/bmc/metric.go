// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"fmt"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func startMetrics() error {
	// u-bmc has been allocated port 9370
	l, err := net.Listen("tcp", "[::]:9370")
	if err != nil {
		return fmt.Errorf("could not listen: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	go func() {
		err := http.Serve(l, mux)
		if err != nil {
			log.Error(err)
		}
	}()
	return nil
}
