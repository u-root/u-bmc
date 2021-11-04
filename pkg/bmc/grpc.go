// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/u-root/u-bmc/config"
	pb "github.com/u-root/u-bmc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

type rpcGpioSystem interface {
	PressButton(context.Context, pb.Button, uint32) (chan bool, error)
}

type rpcFanSystem interface {
	ReadFanPercentage(int) (int, error)
	ReadFanRpm(int) (int, error)
	FanCount() int
}

type rpcUartSystem interface {
	NewReader(<-chan struct{}) <-chan []byte
	NewWriter() chan<- []byte
}

type mgmtServer struct {
	gpio rpcGpioSystem
	fan  rpcFanSystem
	uart rpcUartSystem
	v    *config.Version
}

var (
	tlsCertificateExpiry = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "ubmc",
		Subsystem: "grpc",
		Name:      "certificate_expiry",
		Help:      "UNIX timestamp when the currently loaded TLS certificate expires for the gRPC server",
	})
	tlsCertificateLoaded = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "ubmc",
		Subsystem: "grpc",
		Name:      "certificate_loaded",
		Help:      "Whether the gRPC server has loaded a certificate",
	})
)

func init() {
	prometheus.MustRegister(tlsCertificateExpiry)
	prometheus.MustRegister(tlsCertificateLoaded)
}

func (m *mgmtServer) PressButton(ctx context.Context, r *pb.ButtonPressRequest) (*pb.ButtonPressResponse, error) {
	c, err := m.gpio.PressButton(ctx, r.Button, r.DurationMs)
	if err != nil {
		return nil, err
	}
	// Wait for completion
	<-c
	return &pb.ButtonPressResponse{}, nil
}

func (m *mgmtServer) GetFans(ctx context.Context, _ *pb.GetFansRequest) (*pb.GetFansResponse, error) {
	r := pb.GetFansResponse{}
	for i := 0; i < m.fan.FanCount(); i++ {
		rpm, err := m.fan.ReadFanRpm(i)
		if err != nil {
			return nil, err
		}
		prct, err := m.fan.ReadFanPercentage(i)
		if err != nil {
			return nil, err
		}
		r.Fan = append(r.Fan, &pb.Fan{
			Fan: uint32(i), Percentage: uint32(prct), Rpm: uint32(rpm),
		})
	}
	return &r, nil
}

func (m *mgmtServer) streamIn(stream pb.ManagementService_StreamConsoleServer, done <-chan struct{}) error {
	if m.uart == nil {
		return nil
	}
	r := m.uart.NewReader(done)
	for d := range r {
		err := stream.Send(&pb.ConsoleData{Data: d})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *mgmtServer) StreamConsole(stream pb.ManagementService_StreamConsoleServer) error {
	if m.uart == nil {
		return nil
	}
	done := make(chan struct{})
	go func() {
		err := m.streamIn(stream, done)
		if err != nil {
			log.Error(err)
		}
	}()
	w := m.uart.NewWriter()
	defer close(done)
	defer close(w)
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		w <- in.Data
	}
}

func (m *mgmtServer) GetVersion(ctx context.Context, r *pb.GetVersionRequest) (*pb.GetVersionResponse, error) {
	return &pb.GetVersionResponse{Version: m.v.Version, GitHash: m.v.GitHash}, nil
}

func (m *mgmtServer) EnableRemote(c *tls.Config) error {
	l, err := net.Listen("tcp", ":443")
	if err != nil {
		return fmt.Errorf("could not listen: %v", err)
	}
	m.newServer(l, c)
	return nil
}

func (m *mgmtServer) newServer(l net.Listener, c *tls.Config) {
	opts := []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	}
	if c != nil {
		creds := credentials.NewTLS(c)
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}

	g := grpc.NewServer(opts...)
	pb.RegisterManagementServiceServer(g, m)
	grpc_prometheus.Register(g)
	reflection.Register(g)
	go func() {
		err := g.Serve(l)
		if err != nil {
			log.Error(err)
		}
	}()
}

func startGRPC(gpio rpcGpioSystem, fan rpcFanSystem, uart rpcUartSystem, v *config.Version) (*mgmtServer, error) {
	l, err := net.Listen("tcp", "[::1]:80")
	if err != nil {
		return nil, fmt.Errorf("could not listen: %v", err)
	}

	s := mgmtServer{gpio, fan, uart, v}
	s.newServer(l, nil)

	return &s, nil
}
