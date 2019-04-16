# Copyright 2019 u-root Authors
#
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file

.PHONY: all

DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
ROOT_DIR := ../../../
CROSS_COMPILE ?= arm-none-eabi-
CC := $(CROSS_COMPILE)gcc
OBJCOPY := $(CROSS_COMPILE)objcopy

data.o: data.S $(ROOT_DIR)/boot/zImage.boot $(ROOT_DIR)/boot/platform.dtb.boot $(ROOT_DIR)/boot/loader.cpio.gz
	(cd $(ROOT_DIR); $(CC) -c -o $(DIR)/$@ $(DIR)/$<)

boot.elf: start.S platform_g5.S main.S data.o $(ROOT_DIR)/boot/boot-config.auto.h $(ROOT_DIR)/platform/$(PLATFORM)/boot/config.h 
	$(CC) -Wl,--build-id=none \
		-Wl,-T boot.ld -Wl,--no-dynamic-linker -nostdlib $^ -o $@ \
		-I$(ROOT_DIR)/platform/$(PLATFORM)/boot/ \
		-I$(ROOT_DIR)/boot

boot.bin: boot.elf
	$(OBJCOPY) --only-section=.text -O binary $< $@
