// Copyright 2021 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/u-root/u-bmc/pkg/grpc/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	reflect "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
)

func defaultOpts() []grpc.CallOption {
	return []grpc.CallOption{
		grpc.WaitForReady(false),
	}
}

func newConnection(addr string) *grpc.ClientConn {
	opts := []grpc.DialOption{
		grpc.WithBlock(),
		//TODO(MDr164) Add proper transport credentials
		grpc.WithInsecure(),
	}
	ctx := context.Background()

	conn, err := grpc.DialContext(ctx, addr, opts...)
	if err != nil {
		log.Fatalf("Could not open connection: %v", err)
	}
	return conn
}

func newClient(conn *grpc.ClientConn) proto.ManagementServiceClient {
	return proto.NewManagementServiceClient(conn)
}

func callRPC(client proto.ManagementServiceClient, args []string) {
	switch args[0] {
	case "fans":
		getFans(client)
	case "version":
		getVersion(client)
	case "button":
		pressButton(client, args[1:])
	default:
		log.Error("Unknown service")
	}
}

func getFans(client proto.ManagementServiceClient) {
	resp, err := client.GetFans(context.Background(), &proto.GetFansRequest{}, defaultOpts()...)
	if err != nil {
		log.Fatalf("Failed getting response: %v", err)
	}
	for _, fan := range resp.Fan {
		fmt.Printf("Fan: %d RPM: %d Percentage: %d\n", fan.Fan, fan.Rpm, fan.Percentage)
	}
}

func getVersion(client proto.ManagementServiceClient) {
	resp, err := client.GetVersion(context.Background(), &proto.GetVersionRequest{}, defaultOpts()...)
	if err != nil {
		log.Fatalf("Failed getting response: %v", err)
	}
	fmt.Printf("Version: %s Hash: %s\n", resp.Version, resp.GitHash)
}

func pressButton(client proto.ManagementServiceClient, buttons []string) {
	fmt.Println(buttons)
	var button proto.Button
	switch buttons[0] {
	case "power":
		button = proto.Button_BUTTON_POWER
	case "reset":
		button = proto.Button_BUTTON_RESET
	default:
		button = proto.Button_BUTTON_UNSPEC
	}
	duration, err := strconv.Atoi(buttons[1])
	if err != nil {
		log.Fatalf("Not a valid time in ms: %s", buttons[1])
	}
	resp, err := client.PressButton(context.Background(), &proto.ButtonPressRequest{
		Button:     button,
		DurationMs: uint32(duration),
	}, defaultOpts()...)
	if err != nil {
		log.Fatalf("Failed getting response: %v", err)
	}
	fmt.Printf("Got response: %s", resp.String())
}

func listServices(conn *grpc.ClientConn) []string {
	refClient, err := reflect.NewServerReflectionClient(conn).ServerReflectionInfo(context.Background(), []grpc.CallOption{}...)
	if err != nil {
		log.Fatalf("Failed to create reflection client: %v", err)
	}
	err = refClient.Send(&reflect.ServerReflectionRequest{
		MessageRequest: &reflect.ServerReflectionRequest_ListServices{
			ListServices: "*",
		},
	})
	if err != nil {
		log.Fatalf("Failed sending request: %v", err)
	}

	resp, err := refClient.Recv()
	if err != nil {
		log.Fatalf("Failed to read response: %v", err)
	}
	errResp := resp.GetErrorResponse()
	if errResp != nil {
		log.Fatalf("Got error response code: %d %s", codes.Code(errResp.ErrorCode), errResp.ErrorMessage)
	}

	listResp := resp.GetListServicesResponse()
	if listResp == nil {
		log.Warn("No remote services found!")
		return nil
	}

	serviceNames := make([]string, len(listResp.Service))
	for i, s := range listResp.Service {
		serviceNames[i] = s.Name
	}
	return serviceNames
}
