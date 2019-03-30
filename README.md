# u-bmc

[![Build
Status](https://circleci.com/gh/u-root/u-bmc.svg?style=shield)](https://circleci.com/gh/u-root/u-bmc)
[![Go Report
Card](https://goreportcard.com/badge/github.com/u-root/u-bmc)](https://goreportcard.com/report/github.com/u-root/u-bmc)
[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](https://github.com/u-root/u-bmc/blob/master/LICENSE)

# Description

u-bmc uses u-root to create a Linux OS distribution that is fully open-source.

u-bmc borrows and contributes to [OpenBMC](https://github.com/openbmc/openbmc) which has
similar high-level goals. The main difference is that u-bmc chooses to challenge the industry status quo.
E.g. where OpenBMC uses IPMI, u-bmc uses gRPC.

# Demo

[![asciicast](https://asciinema.org/a/202889.png)](https://asciinema.org/a/202889)

# Why?

BMC software has historically been known to be insecure. There is no inherent reason for that.
u-bmc sets out to improve this by offering an alternative built on modern technologies.

# Support

u-bmc is still in experimental stage and is currently only supporting
BMCs based on ASPEED AST2400. Other BMC SOCs are planned, and if you want
to contribute let us know.

Currently the supported boards are:
- Open Compute Project: Quanta F06 Leopard DDR3

Planned boards are:
- Open Compute Project: Quanta F20 Yosemite

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

# Usage

Prerequisites:
```
sudo apt-get install gcc-arm-none-eabi mtd-utils golang-1.12 fakeroot flex bison device-tree-compiler bc libssl-dev
```

Clone:
```
go get github.com/u-root/u-bmc
cd ~/go/src/github.com/u-root/u-bmc
git submodule init && git submodule update
# Until https://github.com/u-root/u-root/issues/1024 is fixed
go get github.com/u-root/u-root
(cd linux/; ../linux-patches/apply.sh)
```

Setup:
```
# SSH ECDSA public keys does not work for now
cp ~/.ssh/id_rsa.pub ssh_keys.pub
make
```

Since u-bmc uses signed binaries it is important that you back up the
contents of boot/keys/ after building as u-bmc will only accept updates
signed with these keys.

# Hacking

To run the unit tests, run `make test`.

To run the simulator and the integration test you need a special
Qemu from https://github.com/openbmc/qemu. Using the upstream Qemu will
not work predictably.

```
make sim
```

To run the integration tests:
```
export UROOT_QEMU=qemu-system-arm
cd integration
go test
```

If you're using a supported platform and want to try it on your hardware you
can use socflash\_x64 provided by ASPEED like this:
```
echo This is extremely likely to break things as u-bmc is still experimental
sudo ./socflash_x64 of=bmc-backup.img if=flash.img lpcport=0x2e option=glc
```

If you want to quickly upload a new build of u-bmc without updating the kernel,
you can use SCP like this:

```
scp root/bbin/bb my-ubmc:/bb
scp root/bbin/bb.sig my-ubmc:/bb.sig
ssh my-ubmc
# Verify that bb is sane by executing /bb
/bb
# Should return:
# <timestmap> You need to specify which command to invoke.
# Exception: /bbin/bb exited with 1
# [tty], line 1: /bbin/bb
mv /bb /bbin/bb
mv /bb.sig /bbin/bb.sig
# Verify the signature before rebooting
gpgv /etc/u-bmc.pub /bbin/bb.sig /bbin/bb
sync
shutdown -r
```

# Updating Dependencies

Latest released version of dep is required. One easy way, but not that secure,
is to install it using their installation script.

```
wget https://raw.githubusercontent.com/golang/dep/master/install.sh
# Verify that it looks sane
cat install.sh
sh install.sh
dep ensure
```

# Contributions

See [CONTRIBUTING.md](CONTRIBUTING.md)

Since this is an early experiment if this is at all interesting for you or your
company, do reach out in our Slack channel:

- [Slack](https://u-root.slack.com), sign up [here](http://slack.u-root.com/)

