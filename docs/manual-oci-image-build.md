# Manual OCI Image Build (Garden Linux -> ironcore-image) for metal-api machinery

This document describes a **manual** workflow to build Garden Linux artifacts and package them as an OCI image using `ironcore-image`.

The resulting OCI image format is what the metal-api machinery expects: kernel (`vmlinuz`), initramfs (`initrd`), optional `root.squashfs`, and optionally a UKI (`.uki`).

## Prerequisites

- A Linux build host (the commands below assume standard Linux tooling).
- Tools: `git`, `tar`, `cpio`, `xz`, and `ukify` (from `systemd` / `systemd-ukify`, depending on your distro).
- `ironcore-image` binary built from the `ironcore-image` repository (`make build` produces `./bin/ironcore-image`).

## 1) Build Garden Linux artifacts

Clone Garden Linux (this repo contains the `./build` scripts that produce the PXE boot artifacts):

```bash
git clone https://github.com/gardenlinux/gardenlinux
cd gardenlinux
```

Build one of the metal targets (choose the one that fits your use case). Each command runs a build pipeline and writes outputs under `.build/`:

```bash
# "vanilla" metal PXE artifact
./build metal_pxe

# CAPI-focused metal artifact
./build metal-capi

# Gardener-focused metal PXE artifact
./build metal-gardener_pxe
```

What to expect after `./build ...`:

- A `.build/` directory gets created.
- Inside `.build/` you will find one or more target-specific archives (tarballs). Those archives contain the actual boot artifacts used by the metal-api machinery, typically including:
  - `vmlinuz` (Linux kernel)
  - `initrd` (initramfs/initramd)
  - `root.squashfs` (root filesystem image)

### Extract the PXE artifact

In `.build/`, pick the tarball that matches your architecture and Garden Linux version and extract it:

```bash
cd .build
tar -xvzf metal_pxe-amd64-<GARDENLINUX_VERSION>-local.pxe.tar.gz
```

Notes:

- Use `tar -xvzf` for `.tar.gz` archives (gzip).
- If your archive ends with `.tar.xz`, use `tar -xvJf <file>.tar.xz` instead.

After extraction, locate `vmlinuz`, `initrd`, and `root.squashfs` in the extracted directory tree (the exact paths depend on the target and Garden Linux version).

## 2) Optional: Append `root.squashfs` to `initrd`

In some setups it is useful to embed the squashfs into the initramfs by appending an additional xz-compressed cpio archive to the existing `initrd`. This can simplify artifact handling (you ship one initrd blob that already contains the squashfs).

Run the following in the directory where `initrd` and `root.squashfs` are located:

```bash
cp initrd initrd.orig
echo root.squashfs | cpio -H newc -o | xz --check=crc32 >> initrd
```

What this does:

- `echo root.squashfs | cpio -H newc -o` creates a tiny cpio archive (newc format) that contains the file `root.squashfs` (as a payload file).
- `xz --check=crc32` compresses that cpio archive.
- `>> initrd` appends the compressed cpio stream to the end of the existing `initrd` (initramfs).

If you do not want this behavior, keep using the original `initrd` (or restore from `initrd.orig`) and ship `root.squashfs` separately.

## 3) Prepare the kernel command line (`cmdline`)

Create a file named `cmdline` next to your boot artifacts:

```text
initrd=initrd gl.ovl=/:tmpfs gl.live=1 ip=any console=ttyS0,115200 console=tty0 earlyprintk=ttyS0,115200 consoleblank=0 ignition.firstboot=1 ignition.config.url=http://<YOUR_IGNITION_ENDPOINT>/ignition ignition.config.url.append.uuid=true ignition.platform.id=metal rd.break SYSTEMD_SULOGIN_FORCE=1
```

Notes:

- This file becomes the kernel command line used by both PXE flows and the UKI build step below.
- Replace `http://<YOUR_IGNITION_ENDPOINT>/ignition` with your actual Ignition endpoint.
- `rd.break` and `SYSTEMD_SULOGIN_FORCE=1` are typically used for debugging / emergency access. For production images, remove them unless you explicitly need them.

## 4) Build a UKI (Unified Kernel Image) with `ukify`

From the directory containing `vmlinuz`, `initrd`, and `cmdline`:

```bash
ukify build \
  --linux vmlinuz \
  --initrd initrd \
  --stub "/usr/lib/systemd/boot/efi/linuxx64.efi.stub" \
  --cmdline "@cmdline" \
  --output today.uki
```

What this does:

- Produces a single `.uki` artifact that bundles the kernel, initrd, and command line for UEFI-based boot flows.
- The stub path is distro-dependent; adjust `--stub` if your system stores the EFI stub elsewhere.

## 5) Build and push the OCI image with `ironcore-image`

### Build `ironcore-image`

In the `ironcore-image` repository:

```bash
make build
```

This should produce `./bin/ironcore-image`.

### Prepare a standard image reference (placeholders)

Use a versioned tag that encodes the upstream Garden Linux version and the metal flavor/variant. A simple convention that works well in practice is:

```text
<REGISTRY>/<ORG>/<REPO>/gardenlinux:<GARDENLINUX_VERSION>-metal-<FLAVOR>-manual
```

Examples (replace placeholders with real values):

- `ghcr.io/ironcore-dev/os-images/gardenlinux:1877.12-metal-manual`
- `ghcr.io/ironcore-dev/os-images/gardenlinux/capi:1877.12-metal-manual`
- `ghcr.io/ironcore-dev/os-images/gardenlinux/gardener:1877.12-metal-manual`

Suggested `<FLAVOR>` values to keep naming consistent with the Garden Linux build target you used:

- `vanilla` for `./build metal_pxe`
- `capi` for `./build metal-capi`
- `gardener` for `./build metal-gardener_pxe`

### Build the image

Copy the artifacts into the `ironcore-image` repo (or adjust paths accordingly). The example below assumes:

- `bin/vmlinuz`
- `bin/initrd`
- `bin/root.squashfs`
- `bin/today.uki`

Build the OCI image:

```bash
./bin/ironcore-image build \
  --tag <REGISTRY>/<ORG>/<REPO>/gardenlinux:<GARDENLINUX_VERSION>-metal-<FLAVOR>-manual \
  --config arch=amd64,squashfs=bin/root.squashfs,initramfs=bin/initrd,kernel=bin/vmlinuz,uki=bin/today.uki
```

What the `--config` values mean:

- `arch=amd64`: target architecture for the image artifacts.
- `kernel=...`: points to the `vmlinuz` you extracted/built.
- `initramfs=...`: points to the `initrd` you extracted/built (optionally with appended squashfs).
- `squashfs=...`: points to the `root.squashfs` root filesystem image. If you fully embedded `root.squashfs` into `initrd` and do not want to ship it as a separate artifact, omit `squashfs=...`.
- `uki=...`: points to the UKI built via `ukify` (optional, but included here to support UKI boot flows).

### Push the image

Push the built image reference to GHCR:

```bash
./bin/ironcore-image push \
  <REGISTRY>/<ORG>/<REPO>/gardenlinux:<GARDENLINUX_VERSION>-metal-<FLAVOR>-manual \
  --push-sub-manifests
```

`--push-sub-manifests` is commonly used when publishing multi-platform images/manifests; keep it enabled if your workflow expects the sub-manifests to be pushed as well.
