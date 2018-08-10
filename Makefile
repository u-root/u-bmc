# Copyright 2018 u-root Authors
# 
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file

LEB=65408

.PHONY: sim

flash.img: u-boot/u-boot-512.bin ubi.img
	bash -c "dd if=<(cat u-boot/u-boot-512.bin ubi.img /dev/zero) of=$(@) bs=1024 count=32768"

boot/u-boot.boot.img: boot/u-boot.boot
	mkimage -T script -C none -n 'u-boot boot script' -d boot/u-boot.boot boot/u-boot.boot.img

linux/.config: linux.config
	cp -v $(<) $(@)

boot/zImage: linux/.config
	make -C linux/ CROSS_COMPILE=arm-none-eabi- ARCH=arm -j3
	cp linux/arch/arm/boot/zImage boot/

boot/f06c-leopard-ddr3.dtb: platform/f06c-leopard-ddr3.dts
	cpp -nostdinc -I linux/arch/arm/boot/dts/ -I linux/include \
		-undef -x assembler-with-cpp $(<) | dtc -O dtb -o $(@) -

boot.ubifs.img: boot/u-boot.boot.img boot/zImage boot/f06c-leopard-ddr3.dtb boot/u-boot.env
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
	make -C u-boot -j8 CC=arm-linux-gnueabi-gcc LD=arm-linux-gnueabi-ld OBJCOPY=arm-linux-gnueabi-objcopy

u-boot/u-boot-512.bin: u-boot/u-boot.bin
	bash -c "dd if=<(cat $(<) /dev/zero) of=$(@) bs=1024 count=512"

sim: flash.img
	qemu-system-arm -m 256 -M palmetto-bmc -nographic -drive file=$(<),format=raw,if=mtd

u-root:
	go get github.com/u-root/u-root
	go build -o u-root github.com/u-root/u-root

initramfs.cpio: u-root ssh_keys.pub
	make -C cmd/uinit ssh_keys.go
	GOARM=5 GOARCH=arm ./u-root -build=bb -o initramfs.cpio \
		github.com/u-root/u-root/cmds/*/ \
		github.com/u-root/elvish \
		github.com/u-root/u-bmc/cmd/*/

clean:
	\rm -f initramfs.cpio u-root \
	 flash.img u-boot/u-boot.bin u-boot/u-boot-512.bin \
	 root.ubifs.img boot.ubifs.img boot/zImage boot/f06c-leopard-ddr3.dtb \
	 boot/u-boot.boot.img ubi.img
	\rm -fr root/
