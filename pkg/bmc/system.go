// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	crand "crypto/rand"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/u-root/u-bmc/config"
	"github.com/u-root/u-bmc/pkg/bmc/ttime"
	pb "github.com/u-root/u-bmc/proto"
	"golang.org/x/sys/unix"
)

const (
	timeRetryDelay                = 10 * time.Second
	timeRetryDelaySecondsJitter   = 10
	timeRefreshDelay              = 7 * time.Hour
	timeRefreshDelaySecondsJitter = 3600
)

const banner = `
██╗   ██╗      ██████╗ ███╗   ███╗ ██████╗
██║   ██║      ██╔══██╗████╗ ████║██╔════╝
██║   ██║█████╗██████╔╝██╔████╔██║██║
██║   ██║╚════╝██╔══██╗██║╚██╔╝██║██║
╚██████╔╝      ██████╔╝██║ ╚═╝ ██║╚██████╗
 ╚═════╝       ╚═════╝ ╚═╝     ╚═╝ ╚═════╝
 `

var (
	environ       []string
	systemVersion = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "ubmc",
		Subsystem: "system",
		Name:      "version",
		Help:      "u-bmc version metric",
	}, []string{"version"})
	systemHasTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "ubmc",
		Subsystem: "system",
		Name:      "has_trusted_time",
		Help:      "u-bmc has acquired trusted time",
	})
)

func init() {
	environ = append(os.Environ(), "USER=root")
	environ = append(environ, "HOME=/root")
	environ = append(environ, "TZ=UTC")

	prometheus.MustRegister(systemVersion)
	prometheus.MustRegister(systemHasTime)
}

type Platform interface {
	InitializeSystem() error
	HostUart() (string, int)
	GpioPlatform
	FanPlatform
}

func newSshKey() []byte {
	pk, err := ecdsa.GenerateKey(elliptic.P521(), crand.Reader)
	if err != nil {
		panic(err)
	}

	asn1, err := x509.MarshalECPrivateKey(pk)
	if err != nil {
		panic(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: asn1})
}

func createFile(file string, mode os.FileMode, c []byte) error {
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, mode)
	if err != nil {
		return fmt.Errorf("open %s for write failed: %v", file, err)
	}
	defer f.Close()
	if _, err := f.Write(c); err != nil {
		return fmt.Errorf("write %s failed: %v", file, err)
	}
	return nil
}

func intrHandler(cmd *exec.Cmd, exited chan bool) {
	for {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		select {
		case _ = <-c:
			cmd.Process.Signal(os.Interrupt)
		case _ = <-exited:
			return
		}
	}
}

func startSsh(ak []string) error {
	if _, err := os.Stat("/config/ssh_host_ecdsa_key"); os.IsNotExist(err) {
		log.Printf("Generating new SSH server key")
		err := createFile("/config/ssh_host_ecdsa_key", 0400, newSshKey())
		if err != nil {
			return fmt.Errorf("createFile for ssh key: %v", err)
		}
	}
	// The variable authorizedKeys is generated by Makefiles
	createFile("/config/authorized_keys", 0600, []byte(strings.Join(ak, "\n")))
	cmd := exec.Command(
		"/bbin/sshd",
		"--ip", "[::0]",
		"--port", "22",
		"--privatekey", "/config/ssh_host_ecdsa_key",
		"--keys", "/config/authorized_keys")
	cmd.Env = environ
	cmd.Stdin = nil
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("startSsh: %v", err)
	}
	return nil
}

func acquireTime(rs []ttime.RoughtimeServer, ntps []ttime.NtpServer) {
	var tt time.Time
	for {
		t, err := ttime.AcquireTime(rs, ntps)
		if err != nil {
			log.Printf("Failed to acquire trusted time: %v", err)
			j := time.Duration(rand.Intn(timeRetryDelaySecondsJitter)) * time.Second
			delay := timeRetryDelay + j
			log.Printf("Waiting %v before retrying time acquisition", delay)
			time.Sleep(delay)
			continue
		} else {
			tt = *t
			break
		}
	}

	log.Printf("Got trusted time: %v", tt)
	tv := unix.NsecToTimeval(tt.UnixNano())
	if err := unix.Settimeofday(&tv); err != nil {
		log.Printf("Unable to set system time: %v", err)
		return
	}
}

func seedRandomGenerator() {
	b := make([]byte, 8)
	_, err := crand.Read(b)
	if err != nil {
		log.Fatalf("Unable to read random seed, cannot safely continue: %v", err)
	}
	buf := bytes.NewReader(b)
	var seed int64
	if err := binary.Read(buf, NativeEndian(), &seed); err != nil {
		log.Fatalf("Unable to convert random seed, cannot safely continue: %v", err)
	}
	rand.Seed(seed)
}

func backgroundTimeSync(rs []ttime.RoughtimeServer, ntps []ttime.NtpServer) {
	for {
		j := time.Duration(rand.Intn(timeRefreshDelaySecondsJitter)) * time.Second
		delay := timeRefreshDelay + j
		log.Printf("Scheduling time re-sync in %s", delay.String())
		tmr := time.NewTimer(timeRefreshDelay)
		<-tmr.C
		log.Printf("Re-syncing trusted time")
		acquireTime(rs, ntps)
	}
}

func importSystemConfiguration(path string) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("Failed to read system configuration %s: %v", path, err)
		log.Printf("Using default system configuration")
		f = []byte{}
	}

	sysconf := &pb.SystemConfig{}
	if err := proto.UnmarshalText(string(f), sysconf); err != nil {
		log.Printf("Failed to unmarshal system configuration: %v", err)
		log.Printf("Using default system configuration")
	}

	if err := ConfigureNetwork(sysconf.Network); err != nil {
		log.Printf("Failed to configure network: %v", err)
	}
}

func Startup(p Platform) error {
	return StartupWithConfig(p, config.DefaultConfig)
}

func StartupWithConfig(p Platform, c *config.Config) error {
	fmt.Printf("\n")
	fmt.Printf(banner)
	fmt.Printf("Welcome to u-bmc version %s\n\n", c.Version.Version)
	systemVersion.With(prometheus.Labels{"version": c.Version.Version}).Inc()

	// Seed the non-crypto random generator using the crypto one (which is
	// hardware based). The non-crypto generator is used for random back-off
	// timers and such, while the crypto one is used for crypto keys.
	seedRandomGenerator()

	loggers := []io.Writer{os.Stdout}
	lf, err := os.OpenFile("/tmp/u-bmc.log", os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		log.Printf("os.OpenFile u-bmc.log: %v", err)
	} else {
		loggers = append(loggers, lf)
		log.SetOutput(io.MultiWriter(loggers...))
	}

	log.Printf("Loading system configuration")
	importSystemConfiguration("/config/system.textpb")

	timeAcquired := make(chan bool)
	go func() {
		log.Printf("Acquiring trusted time")
		acquireTime(c.RoughtimeServers, c.NtpServers)
		// TODO(bluecmd): If the RTC is already set, we should send this straight away
		timeAcquired <- true
	}()

	createFile("/etc/passwd", 0644, []byte("root:x:0:0:root:/root:/bbin/elvish"))
	createFile("/etc/group", 0644, []byte("root:x:0:"))

	// The platform libraries need access to physical memory
	unix.Mknod("/dev/mem", unix.S_IFCHR|0600, 0x0101)

	if c.StartDebugSshServer {
		log.Printf("Starting debug SSH server")
		// Make sure sshd starts up completely before we continue, to allow for debugging
		if err := startSsh(c.DebugSshServerKeys); err != nil {
			log.Printf("ssh server failed: %v", err)
		}
	}

	log.Printf("Initialize system hardware")
	if err := p.InitializeSystem(); err != nil {
		log.Printf("platform.InitializeSystem: %v", err)
		return err
	}

	log.Printf("Starting GPIO drivers")
	gpio, err := startGpio(p)
	if err != nil {
		log.Printf("startGpio failed: %v", err)
		return err
	}

	log.Printf("Starting fan system")
	fan, err := startFan(p)
	if err != nil {
		log.Printf("startFan failed: %v", err)
		return err
	}

	tty, baud := p.HostUart()
	log.Printf("Configuring host UART console %s @ %d baud", tty, baud)
	uart, err := startUart(tty, baud)
	if err != nil {
		log.Printf("startUart failed: %v", err)
		return err
	}

	log.Printf("Starting OpenMetrics interface")
	if err := startMetrics(); err != nil {
		log.Printf("startMetrics failed: %v", err)
		return err
	}

	log.Printf("Starting gRPC interface")
	rpc, err := startGrpc(gpio, fan, uart, &c.Version)
	if err != nil {
		log.Printf("startGrpc failed: %v", err)
		return err
	}

	go func() {
		// Before we enable remote calls, make sure we have acquired accurate time
		<-timeAcquired
		systemHasTime.Set(1)

		log.Printf("Time has been verified, enabling remote RPCs")
		if err := rpc.EnableRemote(); err != nil {
			log.Printf("rpc.EnableRemote failed: %v", err)
		}

		// Start background time sync
		go backgroundTimeSync(c.RoughtimeServers, c.NtpServers)
	}()

	return nil
}

func Shell() {
	cmd := exec.Command("/bbin/login")
	cmd.Dir = "/"
	cmd.Env = environ
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	exited := make(chan bool)
	// Forward intr to the shell
	go intrHandler(cmd, exited)
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to execute: %v", err)
	}
	exited <- true
}

func Reboot() {
	if err := unix.Reboot(unix.LINUX_REBOOT_CMD_RESTART); err != nil {
		log.Fatalf("reboot failed: %v", err)
	}
}
