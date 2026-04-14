// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package xarchives

import (
	"context"
	"os"
	"path/filepath"

	"github.com/ironcore-dev/os-images/internal/xos"
	"github.com/mholt/archives"
)

func ExtractToDir(dir string) archives.FileHandler {
	return func(ctx context.Context, info archives.FileInfo) error {
		filename := filepath.Join(dir, info.NameInArchive)
		if info.IsDir() {
			return os.MkdirAll(filename, info.Mode())
		}

		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			return err
		}

		// Handle symlinks.
		if info.LinkTarget != "" && info.Mode()&os.ModeSymlink != 0 {
			_ = os.Remove(filename) // tar entries may overwrite earlier ones
			return os.Symlink(info.LinkTarget, filename)
		}

		// Handle hard links.
		if info.LinkTarget != "" {
			_ = os.Remove(filename)
			return os.Link(filepath.Join(dir, info.LinkTarget), filename)
		}

		f, err := info.Open()
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()

		return xos.WriteFileReader(filename, f, info.Mode())
	}
}
