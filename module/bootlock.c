/*
 * Copyright 2018 the u-root Authors. All rights reserved
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 */

#include <linux/kobject.h>
#include <linux/sysfs.h>
#include <linux/module.h>
#include <linux/init.h>
#include <linux/mtd/mtd.h>
#include <linux/mtd/spi-nor.h>
#include <linux/mutex.h>

#define BOOT_AREA 512*1024

#define ASPEED_FMC_AHB_BASE 0x1e620000
#define ASPEED_NOR_AHB_BASE 0x20000000

#define MT25Q_READ_VOLATILE_LOCK 0xe8
#define MT25Q_WRITE_VOLATILE_LOCK 0xe5
#define COMMON_OP_WREN 0x06
#define COMMON_OP_RDID 0x9f
#define COMMON_OP_RDSR 0x05

static struct spi_nor *nor;
static struct spi_nor_controller_ops *ctrl;
static void __iomem* aspeed_fmc_base;
static void __iomem* aspeed_nor_base;

static void aspeed_user_control(int ctrl) {
	uint32_t r;
	r = ioread32(aspeed_fmc_base + 0x10) & ~(0x3);
	r |= ctrl ? 0x3 : 0;
	iowrite32(r, aspeed_fmc_base + 0x10);
}

static void aspeed_cs(int cs) {
	uint32_t r;
	r = ioread32(aspeed_fmc_base + 0x10) & ~(0x4);
	r |= (!cs) << 2;
	iowrite32(r, aspeed_fmc_base + 0x10);
}

static int aspeed_read8(uint32_t addr, uint8_t op) {
	int ret;
	__be32 temp;
	if (nor->addr_width != 4) {
		panic("nor->addr_width not 4");
	}
	temp = cpu_to_be32(addr);
	aspeed_cs(1);
	iowrite8(op, aspeed_nor_base);
	iowrite8_rep(aspeed_nor_base, &temp, 4);
	ret = ioread8(aspeed_nor_base);
	aspeed_cs(0);
	return ret;
}

static void aspeed_write8(uint32_t addr, uint8_t op, uint8_t d) {
	__be32 temp;
	if (nor->addr_width != 4) {
		panic("nor->addr_width not 4");
	}
	temp = cpu_to_be32(addr);
	aspeed_cs(1);
	iowrite8(COMMON_OP_WREN, aspeed_nor_base);
	aspeed_cs(0);
	aspeed_cs(1);
	iowrite8(op, aspeed_nor_base);
	iowrite8_rep(aspeed_nor_base, &temp, 4);
	iowrite8(d, aspeed_nor_base);
	aspeed_cs(0);
}

static int aspeed_busy(void) {
	int sr;
	aspeed_cs(1);
	iowrite8(COMMON_OP_RDSR, aspeed_nor_base);
	sr = ioread8(aspeed_nor_base);
	aspeed_cs(0);
	return sr & 0x1;
}

static int mt25q_read_vol_lock(uint32_t addr) {
	return aspeed_read8(addr, MT25Q_READ_VOLATILE_LOCK);
}

static void mt25q_write_vol_lock(uint32_t addr, uint8_t val) {
	while(aspeed_busy());
	aspeed_write8(addr, MT25Q_WRITE_VOLATILE_LOCK, val & 0x3);
}

static int device_supported(void) {
	int id;
	int sup;
	aspeed_cs(1);
	iowrite8(COMMON_OP_RDID, aspeed_nor_base);
	id = ioread32(aspeed_nor_base);
	switch (id) {
		case 0x20ba20: // MT25QL512ABB
			sup = 1;
			break;
		default:
			sup = 0;
	}
	aspeed_cs(0);
	return sup;
}

static ssize_t lock_show(struct kobject *kobj, struct kobj_attribute *attr,
                         char *buff) {
	int ret;
	int locked = 1;
	uint32_t addr;

	mutex_lock(&nor->lock);
	ctrl->prepare(nor);
	aspeed_user_control(1);

	if (!device_supported()) {
		ret = -EMEDIUMTYPE;
		goto read_err;
	}

	for (addr = 0; addr < BOOT_AREA;) {
		uint8_t r;
		r = mt25q_read_vol_lock(addr);
		// TODO(bluecmd): Remove when proven in the field
		printk(KERN_INFO "read from %08x returned %x\n", addr, r);
		if ((r & 0x3) != 0x3) {
			locked = 0;
			break;
		}
		if (addr < 64*1024) {
			addr += 4 * 1024;
		} else {
			addr += 64 * 1024;
		}
	}

read_err:
	aspeed_user_control(0);
	ctrl->unprepare(nor);
	mutex_unlock(&nor->lock);
	if (ret < 0) {
		return ret;
	} else {
		strncpy(buff, locked ? "1" : "0", 2);
		return 2;
	}
}

// Any write to the lock file will make the flash enter lockdown mode
static ssize_t lock_store(struct kobject *kobj, struct kobj_attribute *attr,
                          const char *buff, size_t count) {
	int ret = 0;
	uint32_t addr;

	mutex_lock(&nor->lock);
	ctrl->prepare(nor);
	aspeed_user_control(1);

	if (!device_supported()) {
		ret = -EMEDIUMTYPE;
		goto lock_err;
	}

	for (addr = 0; addr < BOOT_AREA;) {
		mt25q_write_vol_lock(addr, 0x3);
		if (addr < 64*1024) {
			addr += 4 * 1024;
		} else {
			addr += 64 * 1024;
		}
	}

lock_err:
	aspeed_user_control(0);
	ctrl->unprepare(nor);
	mutex_unlock(&nor->lock);
	return ret < 0 ? ret : count;
}

static struct kobj_attribute bootlock_attribute =
	__ATTR(lock, 0600, lock_show, lock_store);

static struct attribute *attrs[] = {
	&bootlock_attribute.attr,
	NULL,
};

static struct attribute_group attr_group = {
	.attrs = attrs,
};

static struct kobject *kobj;

int __init sysfs_init(void) {
	struct mtd_info *mtd;
	int ret;

	kobj = kobject_create_and_add("bootlock", kernel_kobj);
	if (!kobj)
		return -ENOMEM;

	ret = sysfs_create_group(kobj, &attr_group);
	if (ret)
		kobject_put(kobj);

	mtd = get_mtd_device_nm("bmc");
	if (!mtd) {
		printk(KERN_ERR "bootlock could not find MTD named 'bmc'");
		return -ENOENT;
	}

	if (mtd->type != MTD_NORFLASH) {
		printk(KERN_ERR "MTD named 'bmc' is not a NOR flash");
		return -EINVAL;
	}

	// TODO(bluecmd): This is not very nice, there must be a better way to get
	// a reference to the spi_nor.
	nor = (struct spi_nor*)mtd->priv;

	// TODO(bluecmd): This should all be part of an "aspeed-fmc" module
	aspeed_fmc_base = ioremap(ASPEED_FMC_AHB_BASE, 0x14);
	aspeed_nor_base = ioremap(ASPEED_NOR_AHB_BASE, 0x10);

	return ret;
}

void __exit sysfs_exit(void) {
	kobject_put(kobj);
}

module_init(sysfs_init);
module_exit(sysfs_exit);
MODULE_LICENSE("GPL");
