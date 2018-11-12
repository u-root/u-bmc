package bmc

import (
	"context"
	"log"
	"net"
	"os"
	"runtime"
	"testing"

	pt "github.com/prometheus/client_golang/prometheus/testutil"
	pb "github.com/u-root/u-bmc/proto"
	"google.golang.org/grpc"
)

var (
	addr = ""
	u    = &fakeUart{make(chan []byte), make(chan []byte)}
	us   = newUartSystem(u)
	m    = &mgmtServer{uart: us}
)

type fakeUart struct {
	R chan []byte
	W chan []byte
}

func (u *fakeUart) Read(dst []byte) (int, error) {
	return copy(dst, <-u.R), nil
}

func (u *fakeUart) Write(src []byte) (int, error) {
	u.W <- src
	return len(src), nil
}

func Server() {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatalf("net.Listen: %v", err)
	}
	addr = l.Addr().String()
	log.Printf("Listening on %s", addr)
	g := grpc.NewServer()
	pb.RegisterManagementServiceServer(g, m)
	go g.Serve(l)
}

func NewClient(t *testing.T) (pb.ManagementServiceClient, *grpc.ClientConn) {
	c, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("grpc.Dial: %v", err)
	}
	return pb.NewManagementServiceClient(c), c
}

func TestMain(m *testing.M) {
	Server()
	os.Exit(m.Run())
}

func TestStreamConsoleReceive(t *testing.T) {
	c, conn := NewClient(t)

	sc, err := c.StreamConsole(context.Background())
	if err != nil {
		t.Fatalf("StreamConsole: %v", err)
	}

	expected := "Testing"
	go func() {
		// Wait for connect to be processed
		for {
			us.m.Lock()
			r := len(us.readers)
			us.m.Unlock()
			if r > 0 {
				break
			}
			runtime.Gosched()
		}
		if v := pt.ToFloat64(uartConsumers); v != 1 {
			t.Errorf("Expected UART consumers metric to be 1, was %v", v)
		}
		u.R <- []byte(expected)
	}()

	m, err := sc.Recv()
	if err != nil {
		t.Fatalf("sc.Recv: %v", err)
	}

	conn.Close()

	// Wait for disconnect to be processed
	for {
		us.m.Lock()
		r := len(us.readers)
		us.m.Unlock()
		if r == 0 {
			break
		}
		runtime.Gosched()
	}

	if v := pt.ToFloat64(uartConsumers); v != 0 {
		t.Errorf("Expected UART consumers metric after disconnect to be 0, was %v", v)
	}

	if string(m.Data) != expected {
		t.Fatalf("StreamConsole reported %s when it should have been %s", m.Data, expected)
	}
}

func TestStreamConsoleTransmit(t *testing.T) {
	c, conn := NewClient(t)
	defer conn.Close()

	sc, err := c.StreamConsole(context.Background())
	if err != nil {
		t.Fatalf("StreamConsole: %v", err)
	}

	expected := "Testing"
	go func() {
		err := sc.Send(&pb.ConsoleData{Data: []byte(expected)})
		if err != nil {
			t.Fatalf("sc.Send: %v", err)
		}
	}()

	d := <-u.W
	if string(d) != expected {
		t.Fatalf("UART write was %s when it should have been %s", d, expected)
	}
}
