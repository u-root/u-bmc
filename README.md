# u-bmc

[![License](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](https://github.com/u-root/u-bmc/blob/master/LICENSE)

# Description

u-bmc uses u-root to create a Linux OS distribution that is fully open-source.

# Support

u-bmc is still in experimental stage and is currently only supporting
BMCs based on ASPEED AST2400. Currently the only motherboard supported is the
Quanta F06 Leopard from Open Compute Project.

# Roadmap

This is to give you, the reader, some sense of what we want to create:

 * All function exported over GRPC like:
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
   * Must have: USB ethernet to host, replaces KCS IPMI interface.
   * Cool to have: USB graphics card + mouse + keyboard for KVM

# Usage

Prerequisites:
```
sudo apt-get install gcc-arm-none-eabi mtd-utils u-boot-tools golang-1.10 fakeroot flex bison

# Until u-root vendoring is working properly, also grab:
go get -u github.com/u-root/u-bmc/cmd/uinit
```

Clone:
```
go get github.com/u-root/u-bmc
cd ~/go/src/github.com/u-root/u-bmc
git submodule init && git submodule update
```

Setup:
```
# SSH ECDSA public keys does not work for now
cp ~/.ssh/id_rsa.pub ssh_keys.pub
fakeroot make
```

If you have qemu-system-arm installed:
```
make sim
```

If you're using a supported platform and want to try it on your hardware you
can use socflash\_x64 provided by ASPEED like this:
```
echo This is extremely likely to break things as u-bmc is still experimental
sudo ./socflash_x64 of=bmc-backup.img if=flash.img lpcport=0x2e option=gl
```

# Updating Dependencies

```
Latest released version of dep is required:
curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
dep ensure
```

# Contributions

See [CONTRIBUTING.md](CONTRIBUTING.md)

Since this is an early experiment if this is at all interesting for you or your
company, do reach out in our Slack channel:

- [Slack](https://u-root.slack.com), sign up [here](http://slack.u-root.com/)

