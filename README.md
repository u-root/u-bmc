# u-bmc

[![Build
Status](https://circleci.com/gh/u-root/u-bmc.svg?style=shield)](https://circleci.com/gh/u-root/u-bmc)
[![Go Report
Card](https://goreportcard.com/badge/github.com/u-root/u-bmc)](https://goreportcard.com/report/github.com/u-root/u-bmc)
[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](https://github.com/u-root/u-bmc/blob/master/LICENSE)

# Description

u-bmc uses u-root to create a Linux OS distribution that is fully open-source.

u-bmc borrows and contributes to [OpenBMC](https://github.com/openbmc/openbmc) which has
similar high-level goals. The main difference is that u-bmc chooses to challenge the industry status quo,
e.g. where OpenBMC uses IPMI, u-bmc uses gRPC.

# Attention

This project is currently undergoing some heavy maintenance. Don't use in production yet!

# Demo

[![asciicast](https://asciinema.org/a/202889.png)](https://asciinema.org/a/202889)

# Why?

BMC software has historically been known to be insecure. There is no inherent reason for that.
u-bmc sets out to improve this by offering an alternative built on modern technologies.

# Support

u-bmc is still in experimental stage and is currently only supporting
BMCs based on ASPEED AST2400 and AST2500. Other BMC SOCs are planned, and if you want
to contribute let us know.

Currently the supported boards are:
- Open Compute Project: Quanta F06 Leopard DDR3
- Aspeed AST2500 Evaluation Board

Planned boards are:
- ASRock Rack PAUL
- Nuvoton Poleg BMC NPCM7XX Evaluation Board
- Open Compute Project: Quanta F20 Yosemite
- Tyan Tempest CX S7106

Do you want to become a contributor of a board? Let us know!

# Roadmap

To give you some sense of what we want to create:

 * All function exported over gRPC like:
   * Serial-over-LAN
   * Sensor data
   * iKVM
   * Updating BIOS
   * POST information
 * Implementation of OpenMetrics for Prometheus integration for sensors
 * Offer SSH server for on-platform debugging
   * Support SSH CA-signed certificates to avoid having to upload a bunch of certs
 * USB device emulation
   * Must have: USB storage from image
   * Cool to have: USB ethernet to host, replaces KCS IPMI interface.
   * Cool to have: USB graphics card + mouse + keyboard for KVM
 * Optional WebUI
   * Uses the same API as the gRPC client
   * optional so the BMC can stay lean without loosing functionality

# Usage

Prerequisites:

u-bmc uses the Taskfile build system, install it using their [official installation guide](https://taskfile.dev/#/installation).

Packages needed:
- go (at least 1.17)
- gcc-arm-none-eabi (for arm32)
- gcc-aarch64-linux-gnu (for arm64)
- mtd-utils (for targets using flash)
- erofs-utils (for targets using block devices)
- fakeroot
- flex
- bison
- device-tree-compiler
- bc
- libssl-dev
- libelf-dev
- qemu-kvm

Get them for 32bit via e.g.:
```
sudo apt install gcc-arm-none-eabi mtd-utils golang fakeroot flex bison device-tree-compiler bc libssl-dev libelf-dev qemu-kvm
```

We also need both u-bmc and u-root in our GOPATH so install them with:
```
GO111MODULE=off go get github.com/u-root/u-root
GO111MODULE=off go get github.com/u-root/u-bmc
```
Or use git clone:
```
mkdir $GOPATH/src/github.com/u-root
cd $GOPATH/src/github.com/u-root
git clone https://github.com/u-root/u-root
git clone https://github.com/u-root/u-bmc
```

Setup configuration:
```
# RSA keys will be considered legacy and support will be added again later

cp ~/.ssh/*.pub config/generate/ssh-pubkeys
```

Build image:
```
cp TARGET.tmpl TARGET
```
then uncomment the desired target platform e.g. qemu-virt-a72 in TARGET and run
```
task build
```
which makes u-bmc generate and use a selfsigned cert for TLS.
If you want to use LetsEncrypt you need to agree to their terms.
You can find them at https://letsencrypt.org/repository/

Since u-bmc uses signed binaries it is important that you back up the
contents of build/boot/keys/ after building as u-bmc will only accept updates
signed with these keys.

## Simulator

Trying out u-bmc is easiest using the simulator.
First select a Qemu target in the TARGET file then to launch it, run:

```
# Build Qemu target

task build

# Launch the u-bmc simulator in another terminal

task virtual-bmc -- 64bit

# (Optional, run in another terminal) Launch a local emulated BIOS to produce some data on the UART
# Needs to have u-bmc simulator above running for it to attach correctly.

task virtual-host
```

When simulating the following TCP/IP ports are set up:

 * 6022/tcp: u-bmc SSH
 * 6053/udp: u-bmc DNS (to be removed)
 * 6443/tcp: u-bmc gRPC
 * 6443/tcp: u-bmc OpenMetrics (under /metrics)

When the u-bmc guest tries to access 10.0.2.100 a local service called
ubmc-pebble is started which uses Let's Encrypt's pebble service to generate
an HTTPS certificate. The CA used is located in config/sim-ca.crt.

You can interact with u-bmc running in the simulator by pressing Enter to get a shell
or by using ubmcctl:

```
go install github.com/u-root/u-bmc/cmd/ubmcctl

# The root CA is regenerated every time pebble is started to prevent
# testing to accidentally become production

curl https://localhost:14000/root --cacert config/generate/sim-pebble.crt > root.crt
echo '127.0.1.2	ubmc.local' | sudo tee -a /etc/hosts
SSL_CERT_FILE=root.crt ubmcctl -host ubmc.local:6443
```

If you restart pebble you need to update root.crt.

## Testing

The easiest way to run all unit tests is to run `task test`.

To run the integration tests: `task test`.

If you're using a supported platform and want to try it on your hardware you
can use socflash\_x64 provided by ASPEED like this:
```
echo This is extremely likely to break things as u-bmc is still experimental
sudo ./socflash_x64 of=bmc-backup.img if=flash.img lpcport=0x2e option=glc
```

## Uploading a new version

If you want to quickly upload a new build of u-bmc without updating the kernel,
you can use SCP like this:

```
scp build/rootfs/bin/bb my-ubmc:/bb
scp build/rootfs/bin/bb.sig my-ubmc:/bb.sig
ssh my-ubmc

# Verify that bb is sane by executing /bb
/bb

# Should return:
# <timestmap> You need to specify which command to invoke.
# Exception: /bin/bb exited with 1
# [tty], line 1: /bin/bb

mv /bb /bin/bb
mv /bb.sig /bin/bb.sig

# Verify the signature before rebooting

gpgv /etc/u-bmc.pub /bin/bb.sig /bin/bb
sync
shutdown -r
```

# Contributions

See [CONTRIBUTING.md](CONTRIBUTING.md)

Since this is an early experiment if this is at all interesting for you or your
company, do reach out in our Slack channel:

- [Slack](https://osfw.slack.com), sign up [here](http://slack.u-root.com/)

