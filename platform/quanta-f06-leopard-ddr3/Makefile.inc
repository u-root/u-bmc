# Copyright 2018 u-root Authors
#
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file

SOC := ast2400
# TODO(bluecmd): Read these from the platform Go definition to share
# between tests and simulation
QEMUFLAGS += -m 128 -M palmetto-bmc

# TODO(bluecmd): The real machine has 8MB of VGA RAM but the QEMU platform has
# used to emulate has 16MB.
RAM_SIZE := 0x7000000  # 128 MB - 16 MB (VGA)

sim: flash.sim.img
	$(QEMU) $(QEMUFLAGS)
	stty sane
