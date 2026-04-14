// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package build

import (
	"context"

	"github.com/ironcore-dev/os-images/cli/os-images/common"
	"github.com/ironcore-dev/os-images/internal/debian/kvm"
	kvmconfig "github.com/ironcore-dev/os-images/internal/debian/kvm/config"
	"github.com/ironcore-dev/os-images/internal/debian/metal"
	metalconfig "github.com/ironcore-dev/os-images/internal/debian/metal/config"
	"github.com/ironcore-dev/os-images/internal/tools/mmdebstrap"
	"github.com/ironcore-dev/os-images/internal/tools/sqfstar"
	"github.com/ironcore-dev/os-images/internal/tools/ukify"
)

var DebianMetal = BuilderCommand(
	"debian-metal",
	func(ctx context.Context, prov common.Provider) (*metal.Builder, error) {
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

		return metal.DefaultBuilder(
			mmdebstrapExec,
			sqfstarExec,
			ukifyExec,
		), nil
	},
	func() *metalconfig.Config { return &metalconfig.Config{} },
)

var DebianKVM = BuilderCommand(
	"debian-kvm",
	func(ctx context.Context, prov common.Provider) (*kvm.Builder, error) {
		return kvm.DefaultBuilder(), nil
	},
	func() *kvmconfig.Config { return &kvmconfig.Config{} },
)
