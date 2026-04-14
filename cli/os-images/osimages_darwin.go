// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package osimages

import (
	"fmt"
	"os"
)

var (
	defaultBaseTempDir string
)

func init() {
	userHome, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("Could not determine user home directory for base temp directory: %v", err))
	}

	defaultBaseTempDir = userHome
}
