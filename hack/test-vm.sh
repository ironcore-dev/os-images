#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VM_DIR="$SCRIPT_DIR/.test-vm"

ARCH=""
MEMORY=2048

# --- Tool paths (resolved lazily) ---
MKE2FS=""
MKFS_FAT=""
QEMU_FW_DIR=""

detect_host_arch() {
  case "$(uname -m)" in
    aarch64|arm64) echo "arm64" ;;
    x86_64)        echo "amd64" ;;
    *)             echo "unknown" ;;
  esac
}

file_size_bytes() {
  stat -f%z "$1" 2>/dev/null || stat -c%s "$1"
}

resolve_tools() {
  if [[ "$(uname -s)" == "Darwin" ]]; then
    MKE2FS="$(brew --prefix e2fsprogs)/sbin/mke2fs"
    MKFS_FAT="$(brew --prefix dosfstools)/sbin/mkfs.fat"
    QEMU_FW_DIR="${QEMU_FW_DIR:-$(brew --prefix qemu)/share/qemu}"
  else
    MKE2FS="mke2fs"
    MKFS_FAT="mkfs.fat"
    QEMU_FW_DIR="${QEMU_FW_DIR:-/usr/share/qemu}"
  fi
}

check_command() {
  if ! command -v "$1" &>/dev/null && [[ ! -x "$1" ]]; then
    echo "Error: $1 not found."
    echo "  $2"
    exit 1
  fi
}

check_prerequisites_common() {
  local qemu_bin="$1"
  check_command "$qemu_bin" "Install with: brew install qemu (macOS) or apt install qemu-system (Linux)"
  check_command "$MKE2FS" "Install with: brew install e2fsprogs (macOS) or apt install e2fsprogs (Linux)"
}

check_prerequisites_uefi() {
  check_command "$MKFS_FAT" "Install with: brew install dosfstools (macOS) or apt install dosfstools (Linux)"
  check_command "mcopy" "Install with: brew install mtools (macOS) or apt install mtools (Linux)"
  check_command "mmd" "Install with: brew install mtools (macOS) or apt install mtools (Linux)"
}

select_accel() {
  local arch="$1"
  local host_arch
  host_arch="$(detect_host_arch)"
  local os
  os="$(uname -s)"

  if [[ "$arch" == "arm64" && "$host_arch" == "arm64" && "$os" == "Darwin" ]]; then
    echo "hvf"
  elif [[ "$arch" == "amd64" && "$host_arch" == "amd64" && "$os" == "Linux" ]]; then
    echo "kvm"
  else
    echo "tcg"
  fi
}

qemu_binary() {
  case "$1" in
    arm64) echo "qemu-system-aarch64" ;;
    amd64) echo "qemu-system-x86_64" ;;
  esac
}

console_device() {
  case "$1" in
    arm64) echo "ttyAMA0" ;;
    amd64) echo "ttyS0" ;;
  esac
}

kernel_cmdline() {
  local arch="$1"
  local console
  console="$(console_device "$arch")"
  echo "boot=squashfs console=tty0 console=${console},115200 earlyprintk=${console},115200 consoleblank=0 ignition.firstboot=1 ignition.platform.id=qemu"
}

create_root_disk() {
  local squashfs="$1"
  local output="$VM_DIR/root-disk.img"

  if [[ -f "$output" ]]; then
    echo "Reusing existing root disk: $output"
    return
  fi

  local staging="$VM_DIR/staging"
  mkdir -p "$staging"
  cp "$squashfs" "$staging/root.squashfs"

  local squashfs_bytes
  squashfs_bytes="$(file_size_bytes "$squashfs")"
  local size_mb=$(( (squashfs_bytes * 120 / 100 + 1048575) / 1048576 ))
  [[ "$size_mb" -lt 300 ]] && size_mb=300

  echo "Creating root disk (${size_mb}MB ext4, label=root)..."
  "$MKE2FS" -t ext4 -L root -d "$staging" "$output" "${size_mb}M" >/dev/null 2>&1

  rm -rf "$staging"
}

create_esp_image() {
  local uki="$1"
  local arch="$2"
  local output="$VM_DIR/esp.img"

  if [[ -f "$output" ]]; then
    echo "Reusing existing ESP image: $output"
    return
  fi

  local boot_entry
  case "$arch" in
    arm64) boot_entry="EFI/BOOT/BOOTAA64.EFI" ;;
    amd64) boot_entry="EFI/BOOT/BOOTX64.EFI" ;;
  esac

  local uki_bytes
  uki_bytes="$(file_size_bytes "$uki")"
  local size_mb=$(( (uki_bytes + 2097151) / 1048576 + 2 ))
  [[ "$size_mb" -lt 64 ]] && size_mb=64

  echo "Creating ESP image (${size_mb}MB FAT32)..."
  truncate -s "${size_mb}M" "$output"
  "$MKFS_FAT" -F 32 "$output" >/dev/null 2>&1

  mmd -i "$output" ::EFI
  mmd -i "$output" ::EFI/BOOT
  mcopy -i "$output" "$uki" "::${boot_entry}"
}

setup_uefi_firmware() {
  local arch="$1"

  case "$arch" in
    arm64)
      PFLASH_CODE="$QEMU_FW_DIR/edk2-aarch64-code.fd"
      PFLASH_VARS="$VM_DIR/edk2-vars.fd"
      if [[ ! -f "$PFLASH_VARS" ]]; then
        truncate -s 64M "$PFLASH_VARS"
      fi
      ;;
    amd64)
      PFLASH_CODE="$QEMU_FW_DIR/edk2-x86_64-code.fd"
      PFLASH_VARS="$VM_DIR/edk2-vars.fd"
      if [[ ! -f "$PFLASH_VARS" ]]; then
        cp "$QEMU_FW_DIR/edk2-i386-vars.fd" "$PFLASH_VARS"
      fi
      ;;
  esac

  if [[ ! -f "$PFLASH_CODE" ]]; then
    echo "Error: UEFI firmware not found: $PFLASH_CODE"
    echo "  Install with: brew install qemu (macOS)"
    exit 1
  fi
}

build_qemu_base_cmd() {
  local arch="$1"
  local accel
  accel="$(select_accel "$arch")"
  local qemu
  qemu="$(qemu_binary "$arch")"

  local cpu machine
  case "$arch" in
    arm64) machine="virt"; [[ "$accel" == "hvf" ]] && cpu="host" || cpu="cortex-a57" ;;
    amd64) machine="q35";  [[ "$accel" == "kvm" ]] && cpu="host" || cpu="qemu64"     ;;
  esac

  if [[ "$accel" == "tcg" && "$(detect_host_arch)" != "$arch" ]]; then
    echo "Warning: Cross-architecture emulation via TCG — boot will be slow." >&2
  fi

  QEMU_CMD=(
    "$qemu"
    -machine "$machine"
    -accel "$accel"
    -cpu "$cpu"
    -m "$MEMORY"
    -nographic
    -netdev user,id=net0
    -device virtio-net-pci,netdev=net0
  )
}

# --- Subcommands ---

cmd_direct() {
  if [[ $# -ne 3 ]]; then
    echo "Usage: $0 direct [options] <vmlinuz> <initrd> <root.squashfs>"
    exit 1
  fi

  local vmlinuz="$1" initrd="$2" squashfs="$3"

  for f in "$vmlinuz" "$initrd" "$squashfs"; do
    if [[ ! -f "$f" ]]; then
      echo "Error: file not found: $f"
      exit 1
    fi
  done

  resolve_tools
  local qemu
  qemu="$(qemu_binary "$ARCH")"
  check_prerequisites_common "$qemu"

  mkdir -p "$VM_DIR"
  create_root_disk "$squashfs"
  build_qemu_base_cmd "$ARCH"

  local cmdline
  cmdline="$(kernel_cmdline "$ARCH")"

  QEMU_CMD+=(
    -kernel "$vmlinuz"
    -initrd "$initrd"
    -append "$cmdline"
    -drive "file=$VM_DIR/root-disk.img,format=raw,if=virtio,media=disk"
  )

  echo ""
  echo "Launching QEMU (direct kernel boot, $ARCH)..."
  echo "  Press Ctrl-A X to exit."
  echo ""
  exec "${QEMU_CMD[@]}"
}

cmd_uefi() {
  if [[ $# -ne 2 ]]; then
    echo "Usage: $0 uefi [options] <uki.img> <root.squashfs>"
    exit 1
  fi

  local uki="$1" squashfs="$2"

  for f in "$uki" "$squashfs"; do
    if [[ ! -f "$f" ]]; then
      echo "Error: file not found: $f"
      exit 1
    fi
  done

  resolve_tools
  local qemu
  qemu="$(qemu_binary "$ARCH")"
  check_prerequisites_common "$qemu"
  check_prerequisites_uefi

  mkdir -p "$VM_DIR"
  create_root_disk "$squashfs"
  create_esp_image "$uki" "$ARCH"
  setup_uefi_firmware "$ARCH"
  build_qemu_base_cmd "$ARCH"

  QEMU_CMD+=(
    -drive "if=pflash,format=raw,file=$PFLASH_CODE,readonly=on"
    -drive "if=pflash,format=raw,file=$PFLASH_VARS"
    -drive "file=$VM_DIR/esp.img,format=raw,if=virtio,media=disk"
    -drive "file=$VM_DIR/root-disk.img,format=raw,if=virtio,media=disk"
  )

  echo ""
  echo "Launching QEMU (UEFI boot, $ARCH)..."
  echo "  Press Ctrl-A X to exit."
  echo ""
  exec "${QEMU_CMD[@]}"
}

cmd_clean() {
  if [[ -d "$VM_DIR" ]]; then
    rm -rf "$VM_DIR"
    echo "Removed $VM_DIR"
  else
    echo "Nothing to clean."
  fi
}

# --- Argument parsing ---

usage() {
  cat <<EOF
Usage: $0 <subcommand> [options] <artifacts...>

Boot a local QEMU VM from metal build artifacts.

Subcommands:
  direct <vmlinuz> <initrd> <root.squashfs>  Direct kernel boot
  uefi   <uki.img> <root.squashfs>           UEFI boot via UKI
  clean                                       Remove VM temp files ($VM_DIR)

Options:
  -a, --arch <arm64|amd64>   Target architecture (default: host)
  -m, --memory <MB>          VM memory in MB (default: 2048)
  -h, --help                 Show this help

Examples:
  $0 direct test/arm64/vmlinuz test/arm64/initrd test/arm64/root.squashfs
  $0 uefi test/arm64/uki.img test/arm64/root.squashfs
  $0 uefi --arch amd64 test/amd64/uki.img test/amd64/root.squashfs
  $0 clean
EOF
  exit 1
}

if [[ $# -eq 0 ]]; then
  usage
fi

SUBCMD="$1"
shift

if [[ "$SUBCMD" == "-h" || "$SUBCMD" == "--help" ]]; then
  usage
fi

# Parse options before positional args.
while [[ $# -gt 0 ]]; do
  case "$1" in
    -a|--arch)
      ARCH="$2"
      shift 2
      ;;
    -m|--memory)
      MEMORY="$2"
      shift 2
      ;;
    -h|--help)
      usage
      ;;
    -*)
      echo "Unknown option: $1"
      usage
      ;;
    *)
      break
      ;;
  esac
done

# Default arch to host.
if [[ -z "$ARCH" ]]; then
  ARCH="$(detect_host_arch)"
fi

if [[ "$ARCH" != "arm64" && "$ARCH" != "amd64" ]]; then
  echo "Error: unsupported architecture: $ARCH"
  exit 1
fi

case "$SUBCMD" in
  direct) cmd_direct "$@" ;;
  uefi)   cmd_uefi "$@" ;;
  clean)  cmd_clean ;;
  *)      echo "Unknown subcommand: $SUBCMD"; usage ;;
esac
