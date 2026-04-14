// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package mmdebstrap

import (
	"context"
	"io"
	"os"
	osexec "os/exec"

	"github.com/docker/docker/client"
)

type Executor interface {
	Run(ctx context.Context, suite string, opts Options) (io.ReadCloser, error)
}

// CopyIn describes a file to copy into the chroot via mmdebstrap's copy-in
type CopyIn struct {
	// SrcFilename is a path to an existing file.
	SrcFilename string
	// SrcContent is the source content to use.
	SrcContent []byte
	// Mode is the mode to write the file with. Can only be used in conjunction with SrcContent.
	Mode os.FileMode
	// DstFilename is the filename in the chroot.
	DstFilename string
}

// Hook is an mmdebstrap hook directive.
// Exactly one of Hook or CopyIn must be set.
type Hook struct {
	Hook   string  // raw shell command
	CopyIn *CopyIn // copy-in directive
}

type Options struct {
	Variant       string
	Architectures []string
	// Components is a list of components like `main`, `contrib`, `non-free` and `non-free-firmware` which
	// will be used for all URI-only mirror arguments.
	Components []string
	// Include is a list of packages to install in addition to the packages installed by the selected variant.
	Include        []string
	EssentialHooks []Hook
}

func GetExecutor(
	ctx context.Context,
	getClient func(ctx context.Context) (*client.Client, error),
	getBaseTempDir func(ctx context.Context) (string, error),
) (Executor, error) {
	if _, err := osexec.LookPath("mmdebstrap"); err == nil {
		return DefaultExec(), nil
	}

	c, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	tempDir, err := getBaseTempDir(ctx)
	if err != nil {
		return nil, err
	}

	return DefaultDocker(ctx, c, tempDir)
}
