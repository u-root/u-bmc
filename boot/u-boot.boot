ubifsload 0x40000000 u-boot.env
env import -t 0x40000000
ubifsload 0x40000000 quanta-f06-leopard-ddr3.dtb
ubifsload 0x40008000 zImage
bootz 0x40008000 - 0x40000000
