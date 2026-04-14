#!/bin/sh
# initramfs-tools hook: include squashfs/overlay support in the initrd.
set -e

PREREQ=""
prereqs() { echo "$PREREQ"; }
case $1 in prereqs) prereqs; exit 0;; esac

. /usr/share/initramfs-tools/hook-functions

manual_add_modules squashfs
manual_add_modules overlay
manual_add_modules loop

copy_exec /sbin/blkid
copy_exec /sbin/losetup

# The default klibc mount does not support overlayfs -o options.
# Copy the real util-linux mount under a distinct name so the boot
# script can invoke it without conflicting with the klibc mount.
copy_exec /usr/bin/mount /sbin/mount.util-linux
