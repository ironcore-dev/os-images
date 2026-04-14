// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package builder

import "context"

type Builder[Config any] interface {
	Build(ctx context.Context, arch string, cfg Config, outputDir string) error
}
