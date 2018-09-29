// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/fullstorydev/grpcurl"
	"github.com/golang/protobuf/proto"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
)

const (
	service = "bmc.ManagementService"
)

var (
	host = flag.String("host", "localhost", "Which u-bmc host to connect to")
)

type handler struct {
	stat *status.Status
}

func (*handler) OnResolveMethod(md *desc.MethodDescriptor) {
}

func (*handler) OnSendHeaders(md metadata.MD) {
}

func (*handler) OnReceiveHeaders(md metadata.MD) {
}

func (*handler) OnReceiveResponse(resp proto.Message) {
	t := proto.MarshalTextString(resp)
	if t != "" {
		log.Printf("%v\n", t)
	}
}

func (h *handler) OnReceiveTrailers(stat *status.Status, md metadata.MD) {
	h.stat = stat
}

func main() {
	flag.Parse()

	var (
		target string
		opts   []grpc.DialOption
		creds  credentials.TransportCredentials
	)

	if *host == "localhost" {
		// Connect to localhost:80 unauthenticated and unecrypted
		// This is used to troubleshoot on the BMC. If you have shell access
		// on the BMC you're already authorized to do anything.
		target = fmt.Sprintf("[::1]:80")
	} else {
		// Connect to host:443 using client credentials and server verification
		// TODO(bluecmd): Add grpcurl.ClientTransportCredentials to opts
		target = fmt.Sprintf("%s:443", *host)
	}

	ctx := context.Background()

	c, err := grpcurl.BlockingDial(ctx, "tcp", target, creds, opts...)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer c.Close()

	refClient := grpcreflect.NewClient(ctx, reflectpb.NewServerReflectionClient(c))
	ds := grpcurl.DescriptorSourceFromServer(ctx, refClient)

	if len(flag.Args()) == 0 {
		usage(ds)
	} else {
		call(ctx, ds, c, flag.Args()[0], strings.Join(flag.Args()[1:], " "))
	}
}

func call(ctx context.Context, ds grpcurl.DescriptorSource, c *grpc.ClientConn, method string, text string) {
	method = fmt.Sprintf("%s.%s", service, method)
	// TODO(https://github.com/fullstorydev/grpcurl/issues/51): Support plain text
	sent := false
	rd := func() ([]byte, error) {
		if sent || text == "" {
			return nil, io.EOF
		}
		sent = true
		return []byte(text), nil
	}
	h := &handler{}
	if err := grpcurl.InvokeRpc(ctx, ds, c, method, []string{} /* headers */, h, rd); err != nil {
		log.Fatalf("grpcurl.InvokeRpc(%s) failed: %v", method, err)
	}
	if h.stat.Code() != codes.OK {
		log.Fatalf("RPC returned error code %s: %s\n", h.stat.Code().String(), h.stat.Message())
	}
}

func usage(ds grpcurl.DescriptorSource) {
	methods, err := grpcurl.ListMethods(ds, service)
	if err != nil {
		log.Fatalf("grpcurl.ListMethods(bmc.ManagementService) failed: %v", err)
	}

	for _, m := range methods {
		s := fmt.Sprintf("%s.%s", service, m)
		dsc, err := ds.FindSymbol(s)
		if err != nil {
			log.Fatalf("FindSymbol(%s) failed: %v", s, err)
		}
		mp := dsc.(*desc.MethodDescriptor)
		inType := mp.GetInputType()
		outType := mp.GetOutputType()

		fmt.Printf("Method: %v\n", m)
		fmt.Printf(" Request:\n")
		printMessage(ds, inType, 1 /* depth */)
		fmt.Printf("\n Response:\n")
		printMessage(ds, outType, 1 /* depth */)
		fmt.Printf("\n")
	}
}

func printMessage(ds grpcurl.DescriptorSource, md *desc.MessageDescriptor, depth int) {
	dsc, err := ds.FindSymbol(md.GetFullyQualifiedName())
	if err != nil {
		log.Fatalf("FindSymbol(%s) failed: %v", md.GetFullyQualifiedName(), err)
	}
	mp := dsc.(*desc.MessageDescriptor)
	ml := 0
	for _, f := range mp.GetFields() {
		if ml < len(f.GetName()) {
			ml = len(f.GetName())
		}
	}
	empty := true
	pad := strings.Repeat("  ", depth)
	for _, f := range mp.GetFields() {
		empty = false
		if f.GetType() == dpb.FieldDescriptorProto_TYPE_MESSAGE {
			fmt.Printf("%s%s {\n", pad, f.GetName())
			printMessage(ds, f.GetMessageType(), depth+1)
			fmt.Printf("%s}\n", pad)
			continue
		}
		fmt.Printf(fmt.Sprintf("%%s%%-%ds: ", ml), pad, f.GetName())
		if f.GetType() == dpb.FieldDescriptorProto_TYPE_ENUM {
			printEnum(ds, f.GetEnumType())
		} else if f.GetType() == dpb.FieldDescriptorProto_TYPE_UINT32 {
			fmt.Printf("[number (>= 0)]\n")
		} else if f.GetType() == dpb.FieldDescriptorProto_TYPE_INT32 {
			fmt.Printf("[number (positive or negative)]\n")
		} else if f.GetType() == dpb.FieldDescriptorProto_TYPE_STRING {
			fmt.Printf("[string]\n")
		} else if f.GetType() == dpb.FieldDescriptorProto_TYPE_BYTES {
			fmt.Printf("[bytes]\n")
		} else {
			fmt.Printf("[unknown type :(]\n")
		}
	}
	if empty {
		fmt.Printf("%s(empty)\n", pad)
	}
}

func printEnum(ds grpcurl.DescriptorSource, ed *desc.EnumDescriptor) {
	dsc, err := ds.FindSymbol(ed.GetFullyQualifiedName())
	if err != nil {
		log.Fatalf("FindSymbol(%s) failed: %v", ed.GetFullyQualifiedName(), err)
	}
	s := make([]string, 0)
	for _, e := range dsc.(*desc.EnumDescriptor).GetValues() {
		if e.GetNumber() == 0 {
			// Skip the enum value 0 due to this being the de facto "unknown value"
			// and should never be specified explicitly.
			continue
		}
		s = append(s, e.GetName())
	}
	fmt.Printf("[%s]\n", strings.Join(s, " | "))
}
