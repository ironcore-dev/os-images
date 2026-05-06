// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"context"

	"github.com/ironcore-dev/os-images/cli/os-images/common"
	debiankvm "github.com/ironcore-dev/os-images/internal/debian/kvm"
	debiankvmconfig "github.com/ironcore-dev/os-images/internal/debian/kvm/config"
	debianmetal "github.com/ironcore-dev/os-images/internal/debian/metal"
	debianmetalconfig "github.com/ironcore-dev/os-images/internal/debian/metal/config"
	gardenlinuxkvm "github.com/ironcore-dev/os-images/internal/gardenlinux/kvm"
	gardenlinuxmetal "github.com/ironcore-dev/os-images/internal/gardenlinux/metal"
	"github.com/ironcore-dev/os-images/internal/tools/mmdebstrap"
	"github.com/ironcore-dev/os-images/internal/tools/sqfstar"
	"github.com/ironcore-dev/os-images/internal/tools/ukify"
)

var DebianMetal = BuilderCommand(
	"debian-metal",
	func(ctx context.Context, prov common.Provider) (*debianmetal.Builder, error) {
		mmdebstrapExec, err := mmdebstrap.GetExecutor(ctx, prov.GetClient, prov.GetBaseTempDir)
		if err != nil {
			return nil, err
		}

		sqfstarExec, err := sqfstar.GetExecutor(ctx, prov.GetClient, prov.GetBaseTempDir)
		if err != nil {
			return nil, err
		}

		ukifyExec, err := ukify.GetExecutor(ctx, prov.GetClient)
		if err != nil {
			return nil, err
		}

		return debianmetal.DefaultBuilder(
			mmdebstrapExec,
			sqfstarExec,
			ukifyExec,
		), nil
	},
	func() *debianmetalconfig.Config { return &debianmetalconfig.Config{} },
)

var DebianKVM = BuilderCommand(
	"debian-kvm",
	func(ctx context.Context, prov common.Provider) (*debiankvm.Builder, error) {
		return debiankvm.DefaultBuilder(), nil
	},
	func() *debiankvmconfig.Config { return &debiankvmconfig.Config{} },
)

var GardenlinuxMetal = BuilderCommand(
	"gardenlinux-metal",
	func(ctx context.Context, prov common.Provider) (*gardenlinuxmetal.Builder, error) {
		ukifyExec, err := ukify.GetExecutor(ctx, prov.GetClient)
		if err != nil {
			return nil, err
		}

		return gardenlinuxmetal.DefaultBuilder(ukifyExec), nil
	},
	func() *gardenlinuxmetal.Config {
		return &gardenlinuxmetal.Config{}
	},
)

var GardenlinuxKVM = BuilderCommand(
	"gardenlinux-kvm",
	func(ctx context.Context, prov common.Provider) (*gardenlinuxkvm.Builder, error) {
		return gardenlinuxkvm.DefaultBuilder(), nil
	},
	func() *gardenlinuxkvm.Config {
		return &gardenlinuxkvm.Config{}
	},
)
