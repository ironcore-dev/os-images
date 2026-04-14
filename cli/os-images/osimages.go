// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package osimages

import (
	"github.com/ironcore-dev/os-images/cli/os-images/build"
	"github.com/ironcore-dev/os-images/cli/os-images/common"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	var (
		baseTempDir = defaultBaseTempDir
		prov        = common.NewProvider(&baseTempDir)
	)

	cmd := &cobra.Command{
		Use: "os-images",
	}

	cmd.AddCommand(
		build.Command(prov),
	)

	cmd.PersistentFlags().StringVar(&baseTempDir, "base-temp-dir", baseTempDir, "Base directory in which a temp directory will be set up.")

	cmd.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
		return prov.Close()
	}

	return cmd
}
