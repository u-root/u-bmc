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

#define MT25Q_READ_VOLATILE_LOCK 0xe8
#define MT25Q_WRITE_VOLATILE_LOCK 0xe5

static struct spi_nor *nor;

static ssize_t lock_show(struct kobject *kobj, struct kobj_attribute *attr,
		char *buff)
{
	u8 read_opcode, read_dummy;
	int ret;
	int locked = 1;
	uint32_t addr;

	// TODO(bluecmd): Do this iff the chip is an MT25Q chip
	mutex_lock(&nor->lock);

	read_opcode = nor->read_opcode;
	read_dummy = nor->read_dummy;

	nor->prepare(nor, SPI_NOR_OPS_LOCK);

	nor->read_opcode = MT25Q_READ_VOLATILE_LOCK;
	nor->read_dummy = 0;

	for (addr = 0; addr < BOOT_AREA;) {
		uint8_t r;
		ret = nor->read(nor, addr, 1, &r);
		if (ret == 0) {
			ret = -EIO;
			goto read_err;
		}
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
	nor->unprepare(nor, SPI_NOR_OPS_LOCK);

	nor->read_opcode = read_opcode;
  nor->read_dummy = read_dummy;

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
		const char *buff, size_t count)
{
	u8 program_opcode;
	int ret;
	uint32_t addr;

	// TODO(bluecmd): Do this iff the chip is an MT25Q chip
	mutex_lock(&nor->lock);

	program_opcode = nor->program_opcode;
	nor->prepare(nor, SPI_NOR_OPS_LOCK);
	nor->program_opcode = MT25Q_WRITE_VOLATILE_LOCK;

	for (addr = 0; addr < BOOT_AREA;) {
		uint8_t r = 0x3;
		ret = nor->write(nor, addr, 1, &r);
		if (ret == 0) {
			ret = -EIO;
			goto write_err;
		}
		if (addr < 64*1024) {
			addr += 4 * 1024;
		} else {
			addr += 64 * 1024;
		}
	}

write_err:
	nor->unprepare(nor, SPI_NOR_OPS_LOCK);
	nor->program_opcode = program_opcode;
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

int __init sysfs_init(void)
{
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

	return ret;
}

void __exit sysfs_exit(void)
{
	kobject_put(kobj);
}

module_init(sysfs_init);
module_exit(sysfs_exit);
MODULE_LICENSE("GPL");
