// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package mmdebstrap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// createHookTempFiles materialises CopyIn hooks that carry inline SrcContent
// into real files so mmdebstrap can access them.  parentDir controls where the
// temp directories are created: pass "" to use the OS default ($TMPDIR), or an
// explicit directory that is known to be accessible to the mmdebstrap process
// (e.g. a path mounted into a Docker VM).
func createHookTempFiles(opts *Options, tempDir string) (cleanup func() error, err error) {
	var tempDirs []string //nolint:prealloc

	cleanup = func() error {
		var errs []error
		for _, dir := range tempDirs {
			if err := os.RemoveAll(dir); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}

	for _, hook := range opts.EssentialHooks {
		copyIn := hook.CopyIn
		if copyIn == nil {
			continue
		}

		if copyIn.SrcFilename != "" {
			continue
		}

		// mmdebstrap copy-in preserves the source file's basename, so the
		// temp file must be named exactly as it should appear in the chroot.
		// We split DstFilename into the target directory (for copy-in) and
		// the basename (for the temp file), writing into a fresh temp dir to
		// avoid collisions.
		dstDir, dstBase := filepath.Split(copyIn.DstFilename)

		hookTempDir, err := os.MkdirTemp(tempDir, "essential-hook-*")
		if err != nil {
			_ = cleanup()
			return nil, fmt.Errorf("creating temp dir: %w", err)
		}
		tempDirs = append(tempDirs, hookTempDir)

		srcPath := filepath.Join(hookTempDir, dstBase)
		if err := os.WriteFile(srcPath, copyIn.SrcContent, copyIn.Mode); err != nil {
			_ = cleanup()
			return nil, fmt.Errorf("writing temp file %q: %w", dstBase, err)
		}

		*hook.CopyIn = CopyIn{
			SrcFilename: srcPath,
			DstFilename: dstDir,
		}
	}

	return cleanup, nil
}

// buildArgs constructs the mmdebstrap CLI arguments
func buildArgs(suite string, opts Options) []string {
	args := []string{
		"--variant=minbase",
		"--architectures=" + strings.Join(opts.Architectures, ","),
		"--components=main,non-free-firmware",
		"--include=" + strings.Join(opts.Include, ","),
	}
	for _, hook := range opts.EssentialHooks {
		switch {
		case hook.CopyIn != nil:
			args = append(args, fmt.Sprintf("--essential-hook=copy-in %s %s", hook.CopyIn.SrcFilename, hook.CopyIn.DstFilename))
		case hook.Hook != "":
			args = append(args, fmt.Sprintf("--essential-hook=%s", hook.Hook))
		}
	}
	args = append(args, suite, "-")
	return args
}
