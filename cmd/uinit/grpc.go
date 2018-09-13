// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"log"
	"net"

	pb "github.com/u-root/u-bmc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type mgmtServer struct {

}

func (m *mgmtServer) PressButton(ctx context.Context, r *pb.ButtonPressRequest) (*pb.ButtonPressResponse, error) {
	log.Printf("Request: %v", *r)
	err := PressButton(r.Button, r.DurationMs)
	if err != nil {
		return nil, err
	}
	return &pb.ButtonPressResponse{}, nil
}

func startGrpc() {
	// TODO(bluecmd): Since we have no RNG, no configuration, etc
	// only use http for now
	l, err := net.Listen("tcp", "[::]:80")
	if err != nil {
		log.Fatalf("could not listen: %v", err)
	}

	s := mgmtServer{}

	g := grpc.NewServer()
	pb.RegisterManagementServiceServer(g, &s)
	reflection.Register(g)
	go g.Serve(l)
}
