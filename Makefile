# Copyright 2018 u-root Authors
#
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file

LEB=65408
ARCH ?= arm
CROSS_COMPILE ?= arm-none-eabi-
MAKE_JOBS ?= -j8
PLATFORM ?= quanta-f06-leopard-ddr3

.PHONY: sim

flash.img: u-boot/u-boot-512.bin ubi.img
	( cat $^ ; perl -e 'print chr(0xFF)x1024 while 1' ) \
	| dd bs=1M count=32 iflag=fullblock > $@

boot/keys/u-bmc.key:
	mkdir -p boot/keys/
	openssl genrsa -out $@ 2048

boot/keys/u-bmc.crt: boot/keys/u-bmc.key
	openssl req -batch -new -x509 -key $< -out $@

boot/out/boot.img: boot/keys/u-bmc.key boot/keys/u-bmc.crt boot/zImage boot/$(PLATFORM).dtb boot/sign.its | u-boot/tools/mkimage
	mkdir -p boot/out
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

boot/%.dtb: platform/%.dts
	cpp \
		-nostdinc \
		-I linux/arch/$(ARCH)/boot/dts/ \
		-I linux/include \
		-undef \
		-x assembler-with-cpp \
		$< \
	| dtc -O dtb -o $@ -

boot.ubifs.img: boot/out/boot.img
	mkfs.ubifs -r boot/out -m 1 -e ${LEB} -c 64 -o $(@)

root.ubifs.img: initramfs.cpio
	rm -fr root/
	mkdir -p root/root
	fakeroot sh -c "(cd root/; cpio -idv < ../$(<)); \
		mkfs.ubifs -r root -m 1 -e ${LEB} -c 440 -o $(@)"

ubi.img: root.ubifs.img boot.ubifs.img
	ubinize -vv -o ubi.img -m 1 -p64KiB ubi.cfg

u-boot/.config: u-boot.config
	cp -v u-boot.config u-boot/.config

u-boot/tools/mkimage: u-boot/.config
	$(MAKE) $(MAKE_JOBS) \
		-C u-boot \
		CROSS_COMPILE=$(CROSS_COMPILE) \
		tools

u-boot/u-boot.bin: u-boot/.config boot/keys/u-bmc.crt | boot/out/boot.img
	$(MAKE) $(MAKE_JOBS) \
		-C u-boot \
		EXT_DTB=../boot/$(PLATFORM).dtb \
		CROSS_COMPILE=$(CROSS_COMPILE) \
		u-boot.bin

u-boot/u-boot-512.bin: u-boot/u-boot.bin
	( cat $^ ; perl -e 'print chr(0xFF)x1024 while 1' ) \
	| dd bs=1K count=512 iflag=fullblock > $@

sim: flash.img
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
	 boot/out/boot.img ubi.img
	\rm -fr root/
