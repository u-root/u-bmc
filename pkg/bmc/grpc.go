// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/u-root/u-bmc/config"
	pb "github.com/u-root/u-bmc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type rpcGpioSystem interface {
	PressButton(context.Context, pb.Button, uint32) (chan bool, error)
}

type rpcFanSystem interface {
	ReadFanPercentage(int) (int, error)
	ReadFanRpm(int) (int, error)
	SetFanPercentage(int, int) error
	FanCount() int
}

type rpcUartSystem interface {
	Read() []byte
	Write([]byte)
}

type mgmtServer struct {
	gpio rpcGpioSystem
	fan  rpcFanSystem
	uart rpcUartSystem
	v    *config.Version
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
			Fan: uint32(i), Mode: pb.FanMode_FAN_MODE_PERCENTAGE,
			Percentage: uint32(prct), Rpm: uint32(rpm),
		})
	}
	return &r, nil
}

func (m *mgmtServer) SetFan(ctx context.Context, r *pb.SetFanRequest) (*pb.SetFanResponse, error) {
	if r.Mode != pb.FanMode_FAN_MODE_PERCENTAGE {
		return nil, fmt.Errorf("Specified fan mode not supported")
	}
	err := m.fan.SetFanPercentage(int(r.Fan), int(r.Percentage))
	if err != nil {
		return nil, err
	}
	return &pb.SetFanResponse{}, nil
}

func (m *mgmtServer) streamIn(stream pb.ManagementService_StreamConsoleServer) error {
	for {
		err := stream.Send(&pb.ConsoleData{Data: m.uart.Read()})
		if err != nil {
			return err
		}
	}
}

func (m *mgmtServer) StreamConsole(stream pb.ManagementService_StreamConsoleServer) error {
	go m.streamIn(stream)
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		m.uart.Write(in.Data)
	}
}

func (m *mgmtServer) GetVersion(ctx context.Context, r *pb.GetVersionRequest) (*pb.GetVersionResponse, error) {
	return &pb.GetVersionResponse{Version: m.v.Version, GitHash: m.v.GitHash}, nil
}

func (m *mgmtServer) EnableRemote() error {
	// TODO(bluecmd): Add HTTPS when that is implemented
	l, err := net.Listen("tcp", "[::]:443")
	if err != nil {
		return fmt.Errorf("could not listen: %v", err)
	}
	m.newServer(l)
	return nil
}

func (m *mgmtServer) newServer(l net.Listener) {
	g := grpc.NewServer()
	pb.RegisterManagementServiceServer(g, m)
	reflection.Register(g)
	go g.Serve(l)
}

func startGrpc(gpio rpcGpioSystem, fan rpcFanSystem, uart rpcUartSystem, v *config.Version) (*mgmtServer, error) {
	l, err := net.Listen("tcp", "[::1]:80")
	if err != nil {
		return nil, fmt.Errorf("could not listen: %v", err)
	}

	s := mgmtServer{gpio, fan, uart, v}
	s.newServer(l)

	return &s, nil
}
