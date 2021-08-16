# Linux patches

These are patches that are in the process of being upstreamed but needs to
be patched now to support some critical functionallity. They will be
applied automatically when the Linux kernel tar file is extracted.

## Updating patches

Use `git am` to apply patches, then use the following command to update
the patch files:

```
git format-patch origin -o /path/to/patches
```
