// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"github.com/ironcore-dev/os-images/cli/os-images/common"
	"github.com/spf13/cobra"
)

func Command(prov common.Provider) *cobra.Command {
	cmd := &cobra.Command{
		Use: "build",
	}

	cmd.AddCommand(
		DebianKVM(prov),
		DebianMetal(prov),
	)

	return cmd
}
