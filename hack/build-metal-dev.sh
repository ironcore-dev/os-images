#!/usr/bin/env bash
set -euo pipefail

REGISTRY="ghcr.io/ironcore-dev"

usage() {
  echo "Usage: $0 <baremetal-pxe-asset.tar.xz>"
  echo
  echo "Builds a metal OCI image with UKI from a Garden Linux baremetal PXE"
  echo "release asset and pushes it to the registry."
  echo
  echo "The image name, version, and architecture are derived from the asset"
  echo "filename. A random suffix is appended to the tag."
  echo
  echo "Example:"
  echo "  $0 baremetal-gardener_pxe-amd64-2150.0.0-eb8696b9.tar.xz"
  echo "  -> ${REGISTRY}/gardenlinux/gardener:2150.0.0-metal-amd64-manual-1a2b3c4d"
  echo
  echo "Prerequisites: ironcore-image, ukify, cpio, xz, xxd"
  exit 1
}

if [ $# -ne 1 ]; then
  usage
fi

ASSET="$1"

if [ ! -f "$ASSET" ]; then
  echo "Error: asset not found: $ASSET"
  exit 1
fi

ASSET_FILENAME="$(basename "$ASSET")"
ASSET_BASE="${ASSET_FILENAME%.tar.xz}"

# Detect architecture from the filename.
if [[ "$ASSET_FILENAME" == *-amd64-* ]]; then
  ARCH="amd64"
  STUB_NAME="linuxx64.efi.stub"
elif [[ "$ASSET_FILENAME" == *-arm64-* ]]; then
  ARCH="arm64"
  STUB_NAME="linuxaa64.efi.stub"
else
  echo "Error: could not detect architecture from filename: $ASSET_FILENAME"
  exit 1
fi

# Extract the asset pattern (everything before -$ARCH-) and map to image name.
ASSET_PATTERN="${ASSET_FILENAME%%-${ARCH}-*}"
case "$ASSET_PATTERN" in
  baremetal_pxe)          IMAGE_NAME="gardenlinux" ;;
  baremetal-gardener_pxe) IMAGE_NAME="gardenlinux/gardener" ;;
  baremetal-capi)         IMAGE_NAME="gardenlinux/capi" ;;
  *)
    echo "Error: unknown asset pattern: $ASSET_PATTERN"
    exit 1
    ;;
esac

# Extract version: strip pattern and arch prefix, then drop the trailing hash.
# e.g. "baremetal-gardener_pxe-amd64-2150.0.0-eb8696b9" -> "2150.0.0-eb8696b9" -> "2150.0.0"
VERSION_AND_HASH="${ASSET_BASE#${ASSET_PATTERN}-${ARCH}-}"
VERSION="${VERSION_AND_HASH%-*}"

RANDOM_SUFFIX="$(head -c4 /dev/urandom | xxd -p)"
IMAGE_TAG="${REGISTRY}/${IMAGE_NAME}:${VERSION}-metal-manual-${RANDOM_SUFFIX}"

echo "Architecture: $ARCH"
echo "Version:      $VERSION"
echo "Image:        $IMAGE_TAG"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "Extracting $ASSET_FILENAME"
tar -xf "$ASSET" -C "$TMPDIR"

echo "Extracting PXE tarball"
tar -xzf "$TMPDIR/${ASSET_BASE}.pxe.tar.gz" -C "$TMPDIR"

for BIN in initrd vmlinuz root.squashfs; do
  if [ ! -f "$TMPDIR/$BIN" ]; then
    echo "Error: $BIN not found after extraction"
    exit 1
  fi
done

echo "Extracting EFI stub"
STUB_PATH="usr/lib/systemd/boot/efi/${STUB_NAME}"
tar -xf "$TMPDIR/${ASSET_BASE}.tar" -C "$TMPDIR" "$STUB_PATH"
EFI_STUB="$TMPDIR/$STUB_PATH"

echo "Building initrd-uki (embedding squashfs into initrd)"
cp "$TMPDIR/initrd" "$TMPDIR/initrd-uki"
(cd "$TMPDIR" && echo root.squashfs | cpio -H newc -o | xz --check=crc32) >> "$TMPDIR/initrd-uki"

CMDLINE="initrd=initrd gl.ovl=/:tmpfs gl.live=1 ip=any console=ttyS0,115200 console=tty0 earlyprintk=ttyS0,115200 consoleblank=0 ignition.firstboot=1 ignition.config.url=http://boot.onmetal.de:8083/ignition ignition.config.url.append.uuid=true ignition.platform.id=metal"

echo "Building UKI"
ukify build \
  --linux "$TMPDIR/vmlinuz" \
  --initrd "$TMPDIR/initrd-uki" \
  --stub "$EFI_STUB" \
  --cmdline "$CMDLINE" \
  --output "$TMPDIR/uki.img"

echo "Building image"
ironcore-image build \
  --tag "$IMAGE_TAG" \
  --config "arch=${ARCH},squashfs=$TMPDIR/root.squashfs,initramfs=$TMPDIR/initrd,kernel=$TMPDIR/vmlinuz,uki=$TMPDIR/uki.img"

echo "Pushing image"
ironcore-image push --push-sub-manifests "$IMAGE_TAG"

echo "Done: $IMAGE_TAG"
