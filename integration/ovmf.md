# OVMF

OVMF is used to provide realistic data on the UART line when running u-bmc
in a simulator.

EDK2 is open-source UEFI firmware, governed under a BSD license and can be
redistributed. OVMF is a EDK2 configuration which can run under QEMU.

## Build Notes

- OS: `Ubuntu 16.04.4 LTS xenial`
- Git tag: `vUDK2018`
- GCC version: `gcc (Ubuntu 5.4.0-6ubuntu1~16.04.10) 5.4.0 20160609`
- Build is not reproducible.
- Find instructions at: https://wiki.ubuntu.com/UEFI/EDK2

Parameters:

```
ACTIVE_PLATFORM       = OvmfPkg/OvmfPkgX64.dsc
BUILD_RULE_CONF       = Conf/build_rule.txt
TARGET_ARCH           = X64
TARGET                = DEBUG
TOOL_CHAIN_CONF       = Conf/tools_def.txt
TOOL_CHAIN_TAG        = GCC5
```

## Running in QEMU

```
qemu-system-x86_64 -bios ovmf.rom -nographic -net none
```
