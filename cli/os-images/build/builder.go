// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ironcore-dev/os-images/cli/os-images/common"
	"github.com/ironcore-dev/os-images/internal/builder"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func BuilderCommand[B builder.Builder[Config], Config any](
	name string,
	newBuilder func(ctx context.Context, prov common.Provider) (B, error),
	newConfig func() Config,
) func(prov common.Provider) *cobra.Command {
	return func(prov common.Provider) *cobra.Command {
		var (
			architectures = []string{runtime.GOARCH}
			cfgFile       string
			outputDir     = "."
		)

		cmd := &cobra.Command{
			Use: name,
			RunE: func(cmd *cobra.Command, args []string) error {
				ctx := cmd.Context()
				return BuilderRun(ctx, name, prov, cfgFile, newConfig, newBuilder, architectures, outputDir)
			},
		}

		cmd.Flags().StringVarP(&outputDir, "output-dir", "o", outputDir, "output directory")
		cmd.Flags().StringSliceVarP(&architectures, "architecture", "a", architectures, "Architectures")
		cmd.Flags().StringVarP(&cfgFile, "config", "c", cfgFile, "Path to a configuration file.")

		return cmd
	}
}

func BuilderRun[B builder.Builder[Config], Config any](
	ctx context.Context,
	name string,
	prov common.Provider,
	cfgFile string,
	newConfig func() Config,
	newBuilder func(ctx context.Context, prov common.Provider) (B, error),
	architectures []string,
	outputDir string,
) error {
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		return fmt.Errorf("reading config file %s: %w", cfgFile, err)
	}

	cfg := newConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("decoding config: %w", err)
	}

	bldr, err := newBuilder(ctx, prov)
	if err != nil {
		return fmt.Errorf("creating builder %s: %w", name, err)
	}

	for _, arch := range architectures {
		archOutputDir := filepath.Join(outputDir, arch)
		if err := os.MkdirAll(archOutputDir, 0755); err != nil {
			return fmt.Errorf("[arch %s] creating output directory: %w", arch, err)
		}

		if err := bldr.Build(ctx, arch, cfg, archOutputDir); err != nil {
			return fmt.Errorf("[arch %s] building: %w", arch, err)
		}
	}
	return nil
}
