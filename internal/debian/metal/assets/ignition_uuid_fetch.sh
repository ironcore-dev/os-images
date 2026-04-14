#!/bin/sh
# SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
# SPDX-License-Identifier: Apache-2.0
#
# ignition-uuid-fetch — fetch Ignition config with machine UUID appended to URL.
#
# Parses the kernel cmdline for:
#   ignition.config.url=<base-url>
#   ignition.config.url.append.uuid=true|1
#
# If both are present, reads the machine UUID from DMI, fetches
# <base-url>/<uuid>, and writes the result to /run/ignition/config.ign
# so that ignition-fetch.service finds it already present and skips
# its own fetch.

set -eu

CMDLINE_PATH="/proc/cmdline"
UUID_PATH="/sys/devices/virtual/dmi/id/product_uuid"
CONFIG_PATH="/run/ignition/config.ign"

# Parse a key=value parameter from the kernel cmdline.
cmdline_value() {
    key="$1"
    for arg in $(cat "$CMDLINE_PATH"); do
        case "$arg" in
            "${key}"=*)
                echo "${arg#*=}"
                return
                ;;
        esac
    done
}

base_url=$(cmdline_value "ignition.config.url")
if [ -z "$base_url" ]; then
    echo "ignition-uuid-fetch: no ignition.config.url on cmdline, nothing to do"
    exit 0
fi

uuid=$(tr '[:upper:]' '[:lower:]' < "$UUID_PATH" | tr -d '[:space:]')
if [ -z "$uuid" ]; then
    echo "ignition-uuid-fetch: failed to read UUID from $UUID_PATH" >&2
    exit 1
fi

full_url="${base_url}/${uuid}"
echo "ignition-uuid-fetch: fetching config from ${full_url}"

mkdir -p "$(dirname "$CONFIG_PATH")"
curl -fsSL --retry 5 --retry-delay 2 -o "$CONFIG_PATH" "$full_url"

echo "ignition-uuid-fetch: config written to ${CONFIG_PATH}"
