# Copyright 2019 u-root Authors
#
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file

SOC := ast2500

# TODO(rjoleary): We have no idea how much memory this thing has.
RAM_SIZE := 0x7000000  # 128 MB - 16 MB (VGA)

sim: flash.sim.img
	$(QEMU) -m 512  -M ast2500-evb $(QEMUFLAGS)
	stty sane
