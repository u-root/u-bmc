# Copyright 2018 u-root Authors
#
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file

PLATFORM ?= quanta-f06-leopard-ddr3

LEB := 65408
CROSS_COMPILE ?= arm-none-eabi-
QEMU ?= qemu-system-arm
# Some useful debug flags:
# - in_asm, show ASM as it's being fed into QEMU
# - unimp, show things that the VM tries to do but isn't implemented in QEMU
# Run "make QEMUDEBUGFLAGS='-d help' sim" for more flags
QEMUDEBUGFLAGS ?= -d guest_errors
QEMUFLAGS ?= -display none \
	-drive file=flash.sim.img,format=raw,if=mtd \
	-chardev socket,id=host,path=host.uart,server,nowait \
	-nic user,hostfwd=udp::6053-:53,hostfwd=tcp::6443-:443,hostfwd=tcp::9370-:9370 \
	${QEMUDEBUGFLAGS}
MAKE_JOBS ?= -j8
ABS_ROOT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))/
# This is used to include garbage in the signing process to test verification
# errors in the integration test. It should not be used for any real builds.
TEST_EXTRA_SIGN ?= /dev/null
# Flash image size in MiB
ROM_SIZE ?= 16
# Since the DTB needs to contains the partitions, and the bootloader contains
# the DTB, we have to guess the size of the DTB + the bootloader ahead of time.
# The bootloader for ast2400 is something like 10KiB, and the DTB is 25 KiB.
# Here we give the extra space a total of 100 KiB to have some space.
EXTRA_BOOT_SPACE ?= 102400
GIT_VERSION=$(shell (cd $(ABS_ROOT_DIR); git describe --tags --long --always))

# This is to allow integration tests that build new root filesystems outside
# of the source root
ifeq ($(ABS_ROOT_DIR),$(PWD)/)
ROOT_DIR :=
else
ROOT_DIR := $(ABS_ROOT_DIR)
endif

all: flash.img

include $(ROOT_DIR)platform/$(PLATFORM)/Makefile.inc
include $(ROOT_DIR)platform/$(SOC)/Makefile.inc

.PHONY: sim all linux-menuconfig-% test vars pebble

u-bmc:
	go get
	go build

# Linux tree comes from the OpenBMC branch, not the official kernel.org
LINUX_VERSION	:= 5.10.53
LINUX_DIR	:= linux-$(LINUX_VERSION)
LINUX_TAR	:= linux-$(LINUX_VERSION).tar.gz
LINUX_URL       := https://github.com/openbmc/linux/archive/refs/tags/v$(LINUX_VERSION).tar.gz
LINUX_HASH      := 159c8d1a29e7080020ac83497f78458d9c91da9d16144a3d6dd5eb615a62ad1f

# Make considers these files as intermediate files and removes them.
# Tell make they are important.
.SECONDARY: $(LINUX_DIR)/build/full/.config $(LINUX_DIR)/build/boot/.config

$(LINUX_TAR):
	wget -O "$@" "$(LINUX_URL)"

$(LINUX_DIR)/.valid: $(LINUX_TAR)
	echo "$(LINUX_HASH)  $<" | sha256sum --check
	mkdir -p "$(LINUX_DIR)"
	tar xf "$<" --strip 1 -C "$(LINUX_DIR)"
	touch "$@"

$(LINUX_DIR)/.patched: $(LINUX_DIR)/.valid
	cd $(LINUX_DIR) ; \
	for patch in ../patches/*.patch ; do \
		echo "=> Applying `basename $$patch`" ; \
		git apply "$$patch" || exit 1 ; \
	done
	touch "$@"

boot/boot.bin: boot/zImage.boot boot/loader.cpio.gz boot/platform.dtb.boot boot/boot-config.auto.h $(shell find $(ROOT_DIR)platform/$(SOC)/ -name \*.S -type f) $(ROOT_DIR)platform/$(PLATFORM)/boot/config.h
	$(MAKE) -C platform/$(SOC)/boot boot.bin PLATFORM=$(PLATFORM) CROSS_COMPILE=$(CROSS_COMPILE)
	ln -sf ../platform/$(SOC)/boot/boot.bin $@

boot/kexec:
	# TODO(bluecmd): https://github.com/u-root/u-root/issues/401
	wget https://github.com/bluecmd/tools/raw/master/arm/kexec -O $@
	echo "cda9f2ded9c068be69f95dea11fdbab013de6c6c785a3d2ab259028378c06653  $@" | \
		sha256sum -c
	chmod 755 boot/kexec

u-bmc.img: $(ROOT_DIR)boot/boot.bin ubi.img $(ROOT_DIR)platform/ubmc-flash-layout.dtsi
	dd if=/dev/zero | tr '\000' '\377' | dd iflag=fullblock bs=64k of=$@ \
		count=$$(($$(grep SIZE= platform/ubmc-flash-layout.dtsi | cut -d= -f2) / 65536))
	dd conv=notrunc if=$< of=$@
	cat ubi.img >>$@

flash.img: u-bmc.img
	dd if=/dev/zero | tr '\000' '\377' | dd iflag=fullblock bs=1M count=$(ROM_SIZE) of=$@
	dd conv=notrunc if=$< of=$@

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

platform/ubmc-flash-layout.dtsi: boot/zImage.boot boot/loader.cpio.gz
	go run platform/cmd/flash-layout/main.go --extra $(EXTRA_BOOT_SPACE) $^ > $@

module/%.ko: $(shell find $(ROOT_DIR)module -name \*.c -type f) boot/zImage.boot
	$(MAKE) $(MAKE_JOBS) \
		-C "$(LINUX_DIR)" \
		O=build/boot \
		CROSS_COMPILE=$(CROSS_COMPILE) \
		ARCH=$(ARCH) \
		M=$(ABS_ROOT_DIR)/module \
		modules
	$(LINUX_DIR)/build/boot/scripts/sign-file sha256 \
		$(LINUX_DIR)/build/boot/certs/signing_key.pem \
		$(LINUX_DIR)/build/boot/certs/signing_key.x509 \
		$@

boot/loader.cpio.gz: boot/loader/loader boot/keys/u-bmc.pub module/bootlock.ko boot/kexec
	rm -f boot/loader.cpio.gz
	sh -c "cd boot/loader/; echo loader | cpio -H newc -ov -F ../loader.cpio"
	sh -c "cd boot/keys/; echo u-bmc.pub | cpio -H newc -oAv -F ../loader.cpio"
	sh -c "cd module/; echo bootlock.ko | cpio -H newc -oAv -F ../boot/loader.cpio"
	sh -c "cd boot/; echo kexec | cpio -H newc -oAv -F loader.cpio"
	gzip boot/loader.cpio

# TODO(bluecmd): Change to ECDSA now when u-boot is gone
boot/keys/u-bmc.key:
	mkdir -p boot/keys/
	chmod 700 boot/keys/
	openssl genrsa -out $@ 2048

# Rebuild the Linux config from the one in the platform directory.
# If you change $(SOC) it is probably best to blow away the Linux directory
$(LINUX_DIR)/build/%/.config: platform/$(SOC)/linux.config.% $(LINUX_DIR)/.patched
	@mkdir -p "$(dir $@)"
	cp "$<" "$@"
	$(MAKE) \
		-C "$(LINUX_DIR)" \
		O="../$(dir $@)" \
		CROSS_COMPILE=$(CROSS_COMPILE) \
		ARCH=$(ARCH) \
		olddefconfig

boot/zImage.%: $(LINUX_DIR)/build/%/.config
	$(MAKE) $(MAKE_JOBS) \
		-C "$(LINUX_DIR)" \
		O="../$(dir $<)" \
		CROSS_COMPILE=$(CROSS_COMPILE) \
		ARCH=$(ARCH) \
		all
	cp "$(dir $<)arch/$(ARCH)/boot/zImage" "$@"


# Interactively edit a Linux config.  Don't forget to save it afterwards
linux-menuconfig.%: $(LINUX_DIR)/build/%/.config
	$(MAKE) $(MAKE_JOBS) \
		-C "$(LINUX_DIR)" \
		O="../$(dir $<)" \
		CROSS_COMPILE=$(CROSS_COMPILE) \
		ARCH=$(ARCH) \
		menuconfig
	rm -f $<.old

# After editing the real .config file, run `make linux-saveconfig.full` to store the
# default (stripped down) configuration file back into the source tree
linux-saveconfig.%: $(LINUX_DIR)/build/%/.config
	$(MAKE) $(MAKE_JOBS) \
		-C "$(LINUX_DIR)" \
		O="../$(dir $<)" \
		CROSS_COMPILE=$(CROSS_COMPILE) \
		ARCH=$(ARCH) \
		savedefconfig
	mv "$(dir $<)defconfig" "platform/$(SOC)/linux.config$(suffix $@)"

integration/bzImage: integration/linux.config
	$(MAKE) $(MAKE_JOBS) \
		-C $(LINUX_DIR) O=build/$@/ \
		KCONFIG_CONFIG="$(ABS_ROOT_DIR)$<"
	rm -f $<.old
	cp $(LINUX_DIR)/build/$@/arch/x86/boot/bzImage $@

linux-integration-menuconfig: integration/linux.config
	$(MAKE) $(MAKE_JOBS) \
		-C $(LINUX_DIR)/ O=build/$@/ \
		KCONFIG_CONFIG="$(ABS_ROOT_DIR)$<" \
		menuconfig
	rm -f $<.old

boot/%.dtb.boot.dummy: platform/$(PLATFORM)/%.dts platform/ubmc-flash-layout.dtsi platform/$(PLATFORM)/boot/config.h boot/loader.cpio.gz
	# Construct the DTB first with dummy addresses, and then again with the real
	# ones. This assumes the DTB does not grow, but since it's only addresses
	# that should be fine.
	go run platform/cmd/boot-config/main.go \
		--ram-start $(RAM_START) \
		--ram-size $(RAM_SIZE) \
		--initrd /dev/null \
		--dtb /dev/null > boot/boot-config.auto.h
	cpp \
		-nostdinc \
		-I $(LINUX_DIR)/arch/$(ARCH)/boot/dts/ \
		-I $(LINUX_DIR)/include \
		-I platform/ \
		-I platform/$(PLATFORM)/boot/ \
		-I boot/ \
		-DBOOTLOADER \
		-undef \
		-x assembler-with-cpp \
		$< \
	| dtc -O dtb -o $@ -

boot/%.dtb.boot: platform/$(PLATFORM)/%.dts boot/%.dtb.boot.dummy
	go run platform/cmd/boot-config/main.go \
		--ram-start $(RAM_START) \
		--ram-size $(RAM_SIZE) \
		--initrd boot/loader.cpio.gz \
		--dtb $@.dummy > boot/boot-config.auto.h
	rm -f $@
	cpp \
		-nostdinc \
		-I $(LINUX_DIR)/arch/$(ARCH)/boot/dts/ \
		-I $(LINUX_DIR)/include \
		-I platform/ \
		-I platform/$(PLATFORM)/boot/ \
		-I boot/ \
		-DBOOTLOADER \
		-undef \
		-x assembler-with-cpp \
		$< \
	| dtc -O dtb -o $@.tmp -
	# Verify that the size in fact didn't change
	bash -c '[[ \
		$$(stat --printf="%s" $@.tmp) == \
		$$(stat --printf="%s" $@.dummy) ]]' || \
		(echo DTB changed size, cannot continue! Please file a bug about this; exit 1)
	mv $@.tmp $@

boot/%.dtb.full: platform/$(PLATFORM)/%.dts boot/%.dtb.boot
	cpp \
		-nostdinc \
		-I $(LINUX_DIR)/arch/$(ARCH)/boot/dts/ \
		-I $(LINUX_DIR)/include \
		-I platform/ \
		-I platform/$(PLATFORM)/boot/ \
		-I boot/ \
		-undef \
		-x assembler-with-cpp \
		$< \
	| dtc -O dtb -o $@ -

root.ubifs.img: initramfs.cpio $(ROOT_DIR)boot/zImage.full $(ROOT_DIR)boot/signer/signer $(ROOT_DIR)boot/platform.dtb.full $(ROOT_DIR)proto/system.textpb.default
	rm -fr root/
	mkdir -p root/root root/etc root/boot
	# TOOD(bluecmd): Move to u-bmc system startup
	echo "nameserver 2001:4860:4860::8888" > root/etc/resolv.conf
	echo "nameserver 2606:4700:4700::1111" >> root/etc/resolv.conf
	echo "nameserver 8.8.8.8" >> root/etc/resolv.conf
	echo "::1 localhost" >> root/etc/hosts
	cp -v $(ROOT_DIR)boot/zImage.full root/boot/zImage-$(GIT_VERSION)
	cat $(ROOT_DIR)boot/zImage.full | $(ROOT_DIR)boot/signer/signer > root/boot/zImage-$(GIT_VERSION).gpg
	cp -v $(ROOT_DIR)boot/platform.dtb.full root/boot/platform-$(GIT_VERSION).dtb
	cat $(ROOT_DIR)boot/platform.dtb.full | $(ROOT_DIR)boot/signer/signer > root/boot/platform-$(GIT_VERSION).dtb.gpg
	ln -sf zImage-$(GIT_VERSION) root/boot/zImage
	ln -sf zImage-$(GIT_VERSION).gpg root/boot/zImage.gpg
	ln -sf platform-$(GIT_VERSION).dtb root/boot/platform.dtb
	ln -sf platform-$(GIT_VERSION).dtb.gpg root/boot/platform.dtb.gpg
	cp -v $(ROOT_DIR)boot/keys/u-bmc.pub root/etc/
	ln -sf bbin/bb.gpg root/init.gpg
	mkdir root/config
	cp $(ROOT_DIR)proto/system.textpb.default root/config/system.textpb
	# Rewrite the symlink to a non-absolute to allow non-chrooted following.
	# This is a workaround for the fact that the loader cannot chroot currently.
	ln -sf bbin/bb root/init
	fakeroot sh -c "(cd root/; cpio -idv < ../$(<)) && \
		cat root/bbin/bb $(TEST_EXTRA_SIGN) | \
			$(ROOT_DIR)boot/signer/signer > root/bbin/bb.gpg && \
		mkfs.ubifs -x zlib -r root -R0 -m 1 -e ${LEB} -c 2047 -o $(@)"

ubi.img: root.ubifs.img $(ROOT_DIR)ubi.cfg
	ubinize -vv -o ubi.img -m 1 -p64KiB $(ROOT_DIR)ubi.cfg

flash.sim.img: flash.img
	dd if=/dev/zero | tr '\000' '\377' | dd iflag=fullblock bs=1M count=32 of=$@
	dd conv=notrunc if=$< of=$@

initramfs.cpio: u-bmc ssh_keys.pub config/config.go $(shell find $(ROOT_DIR)cmd $(ROOT_DIR)platform/$(PLATFORM) $(ROOT_DIR)pkg $(ROOT_DIR)proto -name \*.go -type f)
	go generate ./config/
	GOARM=5 GOARCH=$(ARCH) ./u-bmc -o "$@.tmp" -p "$(PLATFORM)"
	mv "$@.tmp" "$@"

test:
	go test $(TESTFLAGS) \
		$(shell find */ -name \*.go | grep -v vendor | cut -f -1 -d '/' | sort -u | xargs -n1 -I{} echo ./{}/... | xargs)

get:
	go get -t $(GETFLAGS) \
		$(shell find */ -name \*.go | grep -v vendor | cut -f -1 -d '/' | sort -u | xargs -n1 -I{} echo ./{}/... | xargs)

vars:
	$(foreach var,$(.VARIABLES),$(info $(var)=$($(var))))

clean:
	\rm -f initramfs.cpio u-root \
	 flash.img flash.sim.img boot/boot-config.auto.h \
	 root.ubifs.img boot/zImage* boot/platform.dtb* \
	 ubi.img boot/loader/loader boot/signer/signer boot/loader.cpio.gz \
	 module/*.o module/*.mod.c module/*.ko module/.*.cmd module/modules.order \
	 module/Module.symvers config/ssh_keys.go config/version.go
	\rm -fr root/ boot/modules/ module/.tmp_versions/ boot/out

pebble:
	go run github.com/letsencrypt/pebble/cmd/pebble \
		-dnsserver 127.0.0.1:6053 \
		-config config/sim-pebble.json

run-ovmf:
	qemu-system-x86_64 -bios $(ROOT_DIR)integration/ovmf.rom \
		-display none \
		-chardev socket,id=host,path=host.uart \
		-serial chardev:host \
		-net none
