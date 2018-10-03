# Copyright 2018 u-root Authors
#
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file

LEB := 65408
ARCH ?= arm
CROSS_COMPILE ?= arm-none-eabi-
MAKE_JOBS ?= -j8
PLATFORM ?= quanta-f06-leopard-ddr3
ROOT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
# This is used to include garbage in the signing process to test verification
# errors in the integration test. It should not be used for any real builds.
TEST_EXTRA_SIGN ?= /dev/null

.PHONY: sim

flash.img: u-boot/u-boot-512.bin ubi.img
	cat $^ > $@

boot/signer/signer: boot/signer/main.go
	go get ./boot/signer/
	go build -o $@ ./boot/signer/

boot/loader/loader: boot/loader/main.go
	go get ./boot/loader/
	GOARM=5 GOARCH=$(ARCH) go build -ldflags="-s -w" -o $@ ./boot/loader/

boot/keys/u-bmc.pub: boot/signer/signer boot/keys/u-bmc.key
	# Run signer to make sure the pub file is created
	echo | boot/signer/signer > /dev/null
	touch boot/keys/u-bmc.pub

boot/loader.cpio.gz: boot/loader/loader boot/keys/u-bmc.pub
	rm -f boot/loader.cpio.gz
	sh -c "cd boot/loader/; echo loader | cpio -H newc -ov -F ../loader.cpio"
	sh -c "cd boot/keys/; echo u-bmc.pub | cpio -H newc -oAv -F ../loader.cpio"
	gzip boot/loader.cpio

boot/keys/u-bmc.key:
	mkdir -p boot/keys/
	chmod 700 boot/keys/
	openssl genrsa -out $@ 2048

boot/keys/u-bmc.crt: boot/keys/u-bmc.key
	openssl req -batch -new -x509 -key $< -out $@

boot.img: boot/keys/u-bmc.key boot/keys/u-bmc.crt boot/zImage boot/$(PLATFORM).dtb boot/sign.its boot/loader.cpio.gz | u-boot/tools/mkimage
	sed "s/PLATFORM/$(PLATFORM)/g" boot/sign.its > boot/sig.its.tmp
	u-boot/tools/mkimage -f boot/sig.its.tmp $@
	u-boot/tools/mkimage \
		-F $@ \
		-k boot/keys/ \
		-K boot/$(PLATFORM).dtb \
		-c $(shell git describe --tags --long) \
		-r
	rm -f boot/sig.its.tmp

boot/zImage: linux.config
	$(MAKE) $(MAKE_JOBS) \
		-C linux/ \
		CROSS_COMPILE=$(CROSS_COMPILE) \
		KCONFIG_CONFIG="../$<" \
		ARCH=$(ARCH)
	cp linux/arch/$(ARCH)/boot/zImage $@

boot/%.dtb: platform/%.dts platform/ubmc-flash-layout.dtsi
	cpp \
		-nostdinc \
		-I linux/arch/$(ARCH)/boot/dts/ \
		-I linux/include \
		-undef \
		-x assembler-with-cpp \
		$< \
	| dtc -O dtb -o $@ -

root.ubifs.img: initramfs.cpio boot.img boot/signer/signer
	rm -fr root/
	mkdir -p root/root root/etc root/boot
	cp -v boot.img root/boot/
	cp -v $(ROOT_DIR)/boot/keys/u-bmc.pub root/etc/
	ln -sf /bbin/bb.sig root/init.sig
	fakeroot sh -c "(cd root/; cpio -idv < ../$(<)) && \
		cat root/bbin/bb $(TEST_EXTRA_SIGN) | \
			$(ROOT_DIR)/boot/signer/signer > root/bbin/bb.sig && \
		mkfs.ubifs -r root -R0 -m 1 -e ${LEB} -c 2047 -o $(@)"

ubi.img: root.ubifs.img
	ubinize -vv -o ubi.img -m 1 -p64KiB ubi.cfg

u-boot/.config: u-boot.config
	cp -v u-boot.config u-boot/.config

u-boot/tools/mkimage: u-boot/.config
	$(MAKE) $(MAKE_JOBS) \
		-C u-boot \
		CROSS_COMPILE=$(CROSS_COMPILE) \
		tools

u-boot/u-boot.bin: u-boot/.config boot/keys/u-bmc.crt | boot.img
	$(MAKE) $(MAKE_JOBS) \
		-C u-boot \
		EXT_DTB=../boot/$(PLATFORM).dtb \
		CROSS_COMPILE=$(CROSS_COMPILE) \
		u-boot.bin

u-boot/u-boot-512.bin: u-boot/u-boot.bin
	( cat $^ ; perl -e 'print chr(0xFF)x1024 while 1' ) \
	| dd bs=1K count=512 iflag=fullblock > $@

flash.sim.img: flash.img
	( cat $^ ; perl -e 'print chr(0xFF)x1024 while 1' ) \
		| dd bs=1M count=32 iflag=fullblock > $@

sim: flash.sim.img
	qemu-system-arm \
		-m 256 \
		-M palmetto-bmc \
		-nographic \
		-drive file=$<,format=raw,if=mtd
	stty sane

u-root:
	go get github.com/u-root/u-root
	go build -o u-root github.com/u-root/u-root

initramfs.cpio: u-root ssh_keys.pub $(shell find . -name \*.go -type f)
	go generate ./config/
	GOARM=5 GOARCH=$(ARCH) ./u-root \
		-build=bb \
		-o "$@.tmp" \
		core \
		github.com/u-root/u-root/cmds/scp/ \
		github.com/u-root/u-root/cmds/sshd/ \
		github.com/u-root/u-bmc/cmd/*/ \
		github.com/u-root/u-bmc/platform/$(PLATFORM)/cmd/*/
	mv "$@.tmp" "$@"

clean:
	\rm -f initramfs.cpio u-root \
	 flash.img u-boot/u-boot.bin u-boot/u-boot-512.bin \
	 root.ubifs.img boot.ubifs.img boot/zImage boot/*.dtb \
	 boot.img ubi.img boot/loader/loader boot/signer/signer
	\rm -fr root/
