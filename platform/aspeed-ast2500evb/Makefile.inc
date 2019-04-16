# Copyright 2019 u-root Authors
#
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file

SOC := ast2500
# The UARTs are in the following order: UART5, VUART, UART1, UART2, UART3, UART4
# TODO(bluecmd): Read these from the platform Go definition to share
# between tests and simulation
QEMUFLAGS += -m 512 -M ast2500-evb \
	-serial mon:stdio \
	-serial null \
	-serial null \
	-serial null \
	-serial chardev:host \
	-serial null

RAM_SIZE := 0x1F000000 # 512 MB - 16 MB (VGA)

sim: flash.sim.img
	$(QEMU) $(QEMUFLAGS)
	stty sane
