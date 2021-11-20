// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grpc

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"

	"github.com/u-root/u-bmc/config"
	"github.com/u-root/u-bmc/pkg/network/web"
	"github.com/u-root/u-bmc/pkg/service/grpc/proto"
	"github.com/u-root/u-bmc/pkg/service/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

type rpcGpioSystem interface {
	PressButton(context.Context, proto.Button, uint32) (chan bool, error)
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
	gpio    rpcGpioSystem
	fan     rpcFanSystem
	uart    rpcUartSystem
	version *config.Version
	proto.UnimplementedManagementServiceServer
}

var (
	log = logger.LogContainer.GetSimpleLogger()
)

func (m *mgmtServer) PressButton(ctx context.Context, r *proto.ButtonPressRequest) (*proto.ButtonPressResponse, error) {
	c, err := m.gpio.PressButton(ctx, r.Button, r.DurationMs)
	if err != nil {
		return nil, err
	}
	// Wait for completion
	<-c
	return &proto.ButtonPressResponse{}, nil
}

func (m *mgmtServer) GetFans(ctx context.Context, _ *proto.GetFansRequest) (*proto.GetFansResponse, error) {
	r := proto.GetFansResponse{}
	for i := 0; i < m.fan.FanCount(); i++ {
		rpm, err := m.fan.ReadFanRpm(i)
		if err != nil {
			return nil, err
		}
		prct, err := m.fan.ReadFanPercentage(i)
		if err != nil {
			return nil, err
		}
		r.Fan = append(r.Fan, &proto.Fan{
			Fan: uint32(i), Percentage: uint32(prct), Rpm: uint32(rpm),
		})
	}
	return &r, nil
}

func (m *mgmtServer) streamIn(stream proto.ManagementService_StreamConsoleServer, done <-chan struct{}) error {
	if m.uart == nil {
		return nil
	}
	r := m.uart.NewReader(done)
	for d := range r {
		err := stream.Send(&proto.ConsoleData{Data: d})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *mgmtServer) StreamConsole(stream proto.ManagementService_StreamConsoleServer) error {
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
	writer := m.uart.NewWriter()
	defer close(done)
	defer close(writer)
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		writer <- in.Data
	}
}

func (m *mgmtServer) GetVersion(ctx context.Context, r *proto.GetVersionRequest) (*proto.GetVersionResponse, error) {
	return &proto.GetVersionResponse{Version: m.version.Version, GitHash: m.version.GitHash}, nil
}

func (m *mgmtServer) EnableRemote(mux *http.ServeMux, c *tls.Config) {
	m.newServer(nil, mux, c)
}

func (m *mgmtServer) newServer(l net.Listener, mux *http.ServeMux, tlsConf *tls.Config) {
	opts := []grpc.ServerOption{
		//TODO(MDr164): Implement subset of go-grpc-middleware for metrics and logger
	}
	if tlsConf != nil {
		creds := credentials.NewTLS(tlsConf)
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}
	gServ := grpc.NewServer(opts...)
	proto.RegisterManagementServiceServer(gServ, m)
	reflection.Register(gServ)
	if l != nil {
		go func() {
			err := gServ.Serve(l)
			if err != nil {
				log.Error(err)
			}
		}()
		return
	}
}

func StartGRPC(serv *web.WebServer, gpio rpcGpioSystem, fan rpcFanSystem, uart rpcUartSystem, v *config.Version) (*mgmtServer, error) {
	s := mgmtServer{gpio, fan, uart, v, proto.UnimplementedManagementServiceServer{}}
	s.newServer(serv.Listener, nil, nil)

	return &s, nil
}
