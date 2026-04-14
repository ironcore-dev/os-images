#!/bin/sh
# initramfs-tools boot script for squashfs root with overlayfs.
# Activated by kernel cmdline: boot=squashfs
#
# Handles two scenarios:
#   1. UKI boot:    squashfs is embedded in the initrd as a CPIO entry (root.squashfs)
#   2. Direct boot: squashfs lives on an ext4 block device labeled "root"

mountroot() {
    . /scripts/functions

    # Load required kernel modules — they are present in the initrd
    # (added by hooks/squashfs) but not auto-loaded.
    modprobe -q loop
    modprobe -q squashfs
    modprobe -q overlay

    sq="/root.squashfs"

    # Case 1: squashfs embedded in initrd (UKI boot).
    # The kernel unpacks all concatenated CPIO archives into the initramfs
    # tmpfs, so the file appears at /root.squashfs.
    if [ ! -f "$sq" ]; then
        # Case 2: squashfs on a labeled block device (direct kernel boot).
        log_begin_msg "Waiting for root device (LABEL=root)"
        slumber=30
        while [ "$slumber" -gt 0 ]; do
            dev=$(blkid -L root 2>/dev/null) && break
            sleep 1
            slumber=$((slumber - 1))
        done
        log_end_msg

        if [ -z "$dev" ]; then
            panic "No block device with LABEL=root found"
        fi

        mkdir -p /mnt/root-dev
        mount -t ext4 -o ro "$dev" /mnt/root-dev
        sq="/mnt/root-dev/root.squashfs"
    fi

    if [ ! -f "$sq" ]; then
        panic "root.squashfs not found"
    fi

    # Mount squashfs read-only.
    # Use losetup explicitly — busybox mount does not handle -o loop.
    # --find --show allocates and attaches atomically, avoiding a race
    # between finding a free device and claiming it.
    loopdev=$(losetup --find --show --read-only "$sq") \
        || panic "Failed to set up loop device"

    mkdir -p /mnt/squashfs
    mount.util-linux -t squashfs -o ro "$loopdev" /mnt/squashfs \
        || panic "Failed to mount squashfs"

    # Set up overlayfs: tmpfs upper + squashfs lower.
    mkdir -p /mnt/overlay
    mount -t tmpfs tmpfs /mnt/overlay
    mkdir -p /mnt/overlay/upper /mnt/overlay/work

    mount.util-linux -t overlay overlay \
        -o lowerdir=/mnt/squashfs,upperdir=/mnt/overlay/upper,workdir=/mnt/overlay/work \
        "${rootmnt}" \
        || panic "Failed to mount overlayfs"

    # Move backing mounts under the new root so run-init can clean the
    # initramfs. Without this, run-init fails with "Directory not empty".
    mkdir -p "${rootmnt}/mnt/squashfs" "${rootmnt}/mnt/overlay"
    mount.util-linux --move /mnt/squashfs "${rootmnt}/mnt/squashfs"
    mount.util-linux --move /mnt/overlay "${rootmnt}/mnt/overlay"
    if [ -d /mnt/root-dev ]; then
        mkdir -p "${rootmnt}/mnt/root-dev"
        mount.util-linux --move /mnt/root-dev "${rootmnt}/mnt/root-dev" 2>/dev/null || true
    fi
}
