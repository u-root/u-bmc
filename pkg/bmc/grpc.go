// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"

	pb "github.com/u-root/u-bmc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type rpcGpioSystem interface {
	PressButton(pb.Button, uint32) error
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
}

func (m *mgmtServer) PressButton(ctx context.Context, r *pb.ButtonPressRequest) (*pb.ButtonPressResponse, error) {
	err := m.gpio.PressButton(r.Button, r.DurationMs)
	if err != nil {
		return nil, err
	}
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

func (m *mgmtServer) streamIn(stream pb.ManagementService_StreamConsoleServer) {
	for {
		stream.Send(&pb.ConsoleData{Data: m.uart.Read()})
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

func startGrpc(gpio rpcGpioSystem, fan rpcFanSystem, uart rpcUartSystem) {
	// TODO(bluecmd): Since we have no RNG, no configuration, etc
	// only use http for now
	l, err := net.Listen("tcp", "[::]:80")
	if err != nil {
		log.Fatalf("could not listen: %v", err)
	}

	s := mgmtServer{gpio, fan, uart}

	g := grpc.NewServer()
	pb.RegisterManagementServiceServer(g, &s)
	reflection.Register(g)
	go g.Serve(l)
}
