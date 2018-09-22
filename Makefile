# Copyright 2018 u-root Authors
#
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file

LEB=65408
ARCH ?= arm
CROSS_COMPILE ?= arm-none-eabi-
MAKE_JOBS ?= -j8

.PHONY: sim

flash.img: u-boot/u-boot-512.bin ubi.img
	( cat $^ ; perl -e 'print chr(0xFF)x1024 while 1' ) \
	| dd bs=1M count=32 iflag=fullblock > $@

boot/u-boot.boot.img: boot/u-boot.boot
	mkimage \
		-T script \
		-C none \
		-n 'u-boot boot script' \
		-d $< \
		$@

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

boot.ubifs.img: boot/u-boot.boot.img boot/zImage boot/quanta-f06-leopard-ddr3.dtb boot/u-boot.env
	mkfs.ubifs -r boot -m 1 -e ${LEB} -c 64 -o $(@)

root: initramfs.cpio
	rm -fr root/
	mkdir -p root/root
	(cd root; cpio -idv < ../$(<))

root.ubifs.img: root
	mkfs.ubifs -r root -m 1 -e ${LEB} -c 440 -o $(@)

ubi.img: root.ubifs.img boot.ubifs.img
	ubinize -vv -o ubi.img -m 1 -p64KiB ubi.cfg

u-boot/.config: u-boot.config
	cp -v u-boot.config u-boot/.config

u-boot/u-boot.bin: u-boot/.config
	$(MAKE) $(MAKE_JOBS) \
		-C u-boot \
		CROSS_COMPILE=$(CROSS_COMPILE)

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

initramfs.cpio: u-root ssh_keys.pub
	$(MAKE) -C cmd/uinit ssh_keys.go
	GOARM=5 GOARCH=$(ARCH) ./u-root \
		-build=bb \
		-o "$@.tmp" \
		core \
		github.com/u-root/u-root/cmds/scp/ \
		github.com/u-root/u-root/cmds/sshd/ \
		github.com/u-root/u-bmc/cmd/*/
	mv "$@.tmp" "$@"

clean:
	\rm -f initramfs.cpio u-root \
	 flash.img u-boot/u-boot.bin u-boot/u-boot-512.bin \
	 root.ubifs.img boot.ubifs.img boot/zImage boot/*.dtb \
	 boot/u-boot.boot.img ubi.img
	\rm -fr root/
