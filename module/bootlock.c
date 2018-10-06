/*
 * Copyright 2018 the u-root Authors. All rights reserved
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 */

#include <linux/kobject.h>
#include <linux/string.h>
#include <linux/sysfs.h>
#include <linux/module.h>
#include <linux/init.h>
#include <linux/jiffies.h>
#include <linux/stat.h>
#include <linux/mutex.h>


static ssize_t lock_show(struct kobject *kobj, struct kobj_attribute *attr,
		char *buff)
{
	strncpy(buff, "hello", 6);
	return 6;
}

static ssize_t lock_store(struct kobject *kobj, struct kobj_attribute *attr,
		const char *buff, size_t count)
{
	return count;
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
	int ret;
	kobj = kobject_create_and_add("bootlock", kernel_kobj);
	if (!kobj)
		return -ENOMEM;

	ret = sysfs_create_group(kobj, &attr_group);
	if (ret)
		kobject_put(kobj);

	return ret;
}

void __exit sysfs_exit(void)
{
	kobject_put(kobj);
}

module_init(sysfs_init);
module_exit(sysfs_exit);
MODULE_LICENSE("GPL");
