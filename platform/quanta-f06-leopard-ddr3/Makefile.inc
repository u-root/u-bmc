# Copyright 2018 u-root Authors
#
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file

SOC := ast2400

# TODO(bluecmd): The real machine has 8MB of VGA RAM but the QEMU platform has
# used to emulate has 16MB.
RAM_SIZE := 0x7000000  # 128 MB - 16 MB (VGA)

sim: flash.sim.img
	qemu-system-arm \
		-m 128 \
		-M palmetto-bmc \
		-nographic \
		-drive file=$<,format=raw,if=mtd \
		-d guest_errors,unimp
	stty sane
