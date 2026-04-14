// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package debian

import (
	_ "embed"
)

// ConsoleDevice returns the serial console device name for the given architecture.
func ConsoleDevice(arch string) string {
	if arch == "arm64" {
		return "ttyAMA0"
	}
	return "ttyS0"
}
