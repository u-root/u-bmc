// Copyright 2018 the u-root Authors. All rights reserved
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bmc

import (
	"bytes"
	crand "crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"time"

	"github.com/cleroux/rtc"
	"github.com/golang/protobuf/proto"
	"github.com/jpillora/backoff"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/u-root/u-bmc/config"
	"github.com/u-root/u-bmc/pkg/bmc/cert"
	"github.com/u-root/u-bmc/pkg/bmc/ttime"
	pb "github.com/u-root/u-bmc/proto"
	"golang.org/x/sys/unix"
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

	timeRetry   = &backoff.Backoff{Min: 1 * time.Second, Max: 1 * time.Hour, Factor: 5, Jitter: true}
	timeRefresh = &backoff.Backoff{Min: 3 * time.Hour, Max: 6 * time.Hour, Factor: 2, Jitter: true}
	certRetry   = &backoff.Backoff{Min: 1 * time.Second, Max: 1 * time.Hour, Factor: 5, Jitter: true}
	certRefresh = &backoff.Backoff{Min: 240 * time.Hour, Max: 480 * time.Hour, Factor: 2, Jitter: true}
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

type RPCServer interface {
	EnableRemote(*tls.Certificate) error
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
		case <-c:
			err := cmd.Process.Signal(os.Interrupt)
			if err != nil {
				log.Error(err)
			}
		case <-exited:
			return
		}
	}
}

func startSsh(ak []string) error {
	_, err := os.Stat("/config/ssh_host_ed25519_key")
	if os.IsNotExist(err) {
		log.Infof("Generating new SSH server key")
		err = SshKeyGen("/config/ssh_host_ed25519_key", "")
		if err != nil {
			log.Errorf("Failed to create server key: %v", err)
		}
	}
	return RunSshServer(ak)
}

func acquireTime(rs []ttime.RoughtimeServer, ntps []ttime.NtpServer) {
	var tt time.Time
	for {
		t, err := ttime.AcquireTime(rs, ntps)
		if err != nil {
			log.Warnf("Failed to acquire trusted time: %v", err)
			delay := timeRetry.Duration()
			log.Warnf("Waiting %v before retrying time acquisition", delay)
			time.Sleep(delay)
			continue
		} else {
			tt = *t
			break
		}
	}
	timeRetry.Reset()

	log.Infof("Got trusted time: %v", tt)
	tv := unix.NsecToTimeval(tt.UnixNano())
	if err := unix.Settimeofday(&tv); err != nil {
		log.Errorf("Unable to set system time: %v", err)
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
		delay := timeRefresh.Duration()
		log.Infof("Scheduling time re-sync in %s", delay.String())
		tmr := time.NewTimer(delay)
		<-tmr.C
		log.Infof("Re-syncing trusted time")
		acquireTime(rs, ntps)
	}
}

func loadSysconf(path string) *pb.SystemConfig {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		log.Warnf("Failed to read system configuration %s: %v", path, err)
		log.Warnf("Using default system configuration")
		f = []byte{}
	}

	sysconf := &pb.SystemConfig{}
	if err := proto.UnmarshalText(string(f), sysconf); err != nil {
		log.Warnf("Failed to unmarshal system configuration: %v", err)
		log.Warnf("Using default system configuration")
	}

	return sysconf
}

func Startup(p Platform) (error, chan error) {
	return StartupWithConfig(p, config.DefaultConfig)
}

func StartupWithConfig(p Platform, c *config.Config) (error, chan error) {
	fmt.Print("\n" + banner)
	fmt.Printf("Welcome to u-bmc version %s\n\n", c.Version.Version)
	systemVersion.With(prometheus.Labels{"version": c.Version.Version}).Inc()

	// Seed the non-crypto random generator using the crypto one (which is
	// hardware based). The non-crypto generator is used for random back-off
	// timers and such, while the crypto one is used for crypto keys.
	seedRandomGenerator()

	log.Infof("Initialize system hardware")
	if err := p.InitializeSystem(); err != nil {
		log.Errorf("platform.InitializeSystem: %v", err)
		return err, nil
	}

	log.Infof("Starting GPIO drivers")
	gpio, err := startGpio(p)
	if err != nil {
		log.Errorf("startGpio failed: %v", err)
		return err, nil
	}

	log.Infof("Starting fan system")
	fan, err := startFan(p)
	if err != nil {
		log.Errorf("startFan failed: %v", err)
		return err, nil
	}

	tty, baud := p.HostUart()
	log.Infof("Configuring host UART console %s @ %d baud", tty, baud)
	uart, err := startUart(tty, baud)
	if err != nil {
		log.Errorf("startUart failed: %v", err)
		return err, nil
	}

	log.Infof("Loading system configuration")
	sysconf := loadSysconf("/config/system.textpb")

	network, err := startNetwork(sysconf.Network)
	if err != nil {
		log.Errorf("startNetwork failed: %v", err)
		return err, nil
	}

	// At this time we can assume having a hostname and network connectivity

	timeAcquired := make(chan bool)
	go func() {
		log.Infof("Acquiring trusted time")
		if time.Now().Year() > 2018 {
			// Consider the RTC trusted.
			// This means that if the RTC is set, don't block waiting getting trusted
			// time. If we do get a new trusted time however, make sure to update RTC.
			timeAcquired <- true
			acquireTime(c.RoughtimeServers, c.NtpServers)
		} else {
			acquireTime(c.RoughtimeServers, c.NtpServers)
			timeAcquired <- true
		}
		r, err := rtc.NewRTC("/dev/rtc0")
		if err != nil {
			log.Errorf("Failed to open RTC: %v", err)
			return
		}
		defer r.Close()
		tu := time.Now().UTC()
		if err := r.SetTime(tu); err != nil {
			log.Errorf("Failed to update RTC: %v", err)
		}
	}()

	err = createFile("/etc/passwd", 0644, []byte("root:x:0:0:root:/root:/bin/elvish"))
	if err != nil {
		log.Error(err)
	}
	err = createFile("/etc/group", 0644, []byte("root:x:0:"))
	if err != nil {
		log.Error(err)
	}

	if c.StartDebugSshServer {
		log.Infof("Starting debug SSH server")
		// Make sure sshd starts up completely before we continue, to allow for debugging
		if err := startSsh(c.DebugSshServerKeys); err != nil {
			log.Errorf("ssh server failed: %v", err)
		}
	}

	log.Infof("Starting OpenMetrics interface")
	if err := startMetrics(); err != nil {
		log.Errorf("startMetrics failed: %v", err)
		return err, nil
	}

	log.Infof("Starting gRPC interface")
	rpc, err := startGRPC(gpio, fan, uart, &c.Version)
	if err != nil {
		log.Errorf("startGRPC failed: %v", err)
		return err, nil
	}

	log.Infof("Starting DNS interface")
	dns, err := startDNS(network.FQDN(), network)
	if err != nil {
		log.Errorf("startDNS failed: %v", err)
		return err, nil
	}

	akey, err := cert.LoadOrGenerateKey("/config/acme.key")
	if err != nil {
		log.Errorf("Failed to load ACME key: %v", err)
		return err, nil
	}

	cm := &cert.Manager{
		FQDN:         network.FQDN(),
		AccountKey:   akey,
		ACMEConfig:   &c.ACME,
		ACMEHandlers: []cert.ACMEHandler{dns},
	}

	// The rest of the startup depends on the system having the correct time,
	// so initialize the rest in the background
	startupResult := make(chan error)
	go func() {
		if err := asyncStartup(p, c, rpc, cm, timeAcquired); err != nil {
			startupResult <- err
			return
		}
		startupResult <- nil
	}()

	return nil, startupResult
}

func asyncStartup(p Platform, c *config.Config, rpc RPCServer, cm *cert.Manager, t chan bool) error {
	// Before we enable remote calls, make sure we have acquired accurate time
	<-t
	systemHasTime.Set(1)

	// Start background time sync
	go backgroundTimeSync(c.RoughtimeServers, c.NtpServers)

	log.Infof("Time has been verified, loading system certificate")
	domain := cm.FQDN
	cf := fmt.Sprintf("/config/%s.crt", domain)
	kf := fmt.Sprintf("/config/%s.key", domain)

	var kp *tls.Certificate
	for {
		skp, err := tls.LoadX509KeyPair(cf, kf)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Errorf("tls.LoadX509KeyPair failed: %v", err)
			}
		}

		kp, err = cm.MaybeRenew(&skp)
		if err == nil {
			break
		}
		log.Errorf("cert.Load failed: %v", err)
		delay := certRetry.Duration()
		log.Infof("Waiting %v before retrying certificate acquisition", delay)
		time.Sleep(delay)
	}
	certRetry.Reset()

	// Fail soft on failing to write these files. We can still serve from memory
	// and next reboot we will have to renew instead.
	if err := cert.SaveKeyPair(kp, cf, kf); err != nil {
		log.Warnf("WARNING: System will use certificate from memory, but will have to renew it on reboot. Error was: %v", err)
	}

	log.Infof("Certificate available, enabling remote RPCs")
	if err := rpc.EnableRemote(kp); err != nil {
		log.Errorf("rpc.EnableRemote failed: %v", err)
		return err
	}

	return nil
}

func Shell() {
	cmd := exec.Command("/bin/login")
	cmd.Dir = "/"
	cmd.Env = environ
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	exited := make(chan bool)
	// Forward intr to the shell
	go intrHandler(cmd, exited)
	if err := cmd.Run(); err != nil {
		log.Errorf("Failed to execute: %v", err)
	}
	exited <- true
}

func Reboot() {
	if err := unix.Reboot(unix.LINUX_REBOOT_CMD_RESTART); err != nil {
		log.Fatalf("reboot failed: %v", err)
	}
}
