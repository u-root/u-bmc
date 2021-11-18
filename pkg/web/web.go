// Copyright 2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package web

import (
	"crypto/tls"
	"net"
	"net/http"
	"strings"
)

// WebServer is the struct that holds all necessary information
// for a single port on which web services are served on
type WebServer struct {
	Mux      *http.ServeMux
	Serv     *http.Server
	Listener net.Listener
}

// NewWebserver returns a pointer to a new WebServer struct and
// initialises it with a new http.ServeMux
func NewWebserver() *WebServer {
	return &WebServer{
		Mux:      http.NewServeMux(),
		Serv:     nil,
		Listener: nil,
	}
}

// SetServer fills the WebServer struct with meaningfull information
// and starts a net.Listener on the provided port. If no port is
// provided one is choosen randomly
func (w *WebServer) SetServer(addr, port string, tlsConf *tls.Config) error {
	w.Serv = &http.Server{
		Addr:      addr + ":" + port,
		Handler:   w.Mux,
		TLSConfig: tlsConf,
	}
	var err error
	w.Listener, err = net.Listen("tcp", addr+":"+port)
	return err
}

// Serve calls http.Serve with the information found in the
// WebServer struct
func (w *WebServer) Serve() {
	if strings.HasSuffix(w.Listener.Addr().String(), "443") {
		//TODO(MDr164): Handle non-selfsigned as well
		http.ServeTLS(w.Listener, w.Mux, "/config/acme/selfsigned.crt", "/config/acme/selfsigned.key")
	} else {
		http.Serve(w.Listener, w.Mux)
	}
}
