// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package mmdebstrap

import (
	"context"
	"io"
	osexec "os/exec"

	"github.com/ironcore-dev/os-images/internal/ulog"
	"github.com/ironcore-dev/os-images/internal/xio"
)

type exec struct {
	executable string
}

func NewExec(executable string) Executor {
	return &exec{executable}
}

func DefaultExec() Executor {
	return NewExec("mmdebstrap")
}

func (e *exec) Run(ctx context.Context, suite string, opts Options) (io.ReadCloser, error) {
	log := ulog.FromContext(ctx)

	cleanupHookTempFiles, err := createHookTempFiles(&opts, "")
	if err != nil {
		return nil, err
	}

	args := buildArgs(suite, opts)

	cmd := osexec.CommandContext(ctx, e.executable, args...)
	rc, err := cmd.StdoutPipe()
	if err != nil {
		_ = cleanupHookTempFiles()
		return nil, err
	}

	group := log.Group("mmdebstrap")
	groupWriter := ulog.GroupWriter(group)

	cmd.Stderr = groupWriter
	if err := cmd.Start(); err != nil {
		group.Fail(err, "Error")
		return nil, err
	}

	return xio.ReaderAndCloser(rc, xio.JoinCloser(
		rc,
		xio.CloserFunc(func() error {
			defer func() { _ = cleanupHookTempFiles() }()
			defer func() { _ = groupWriter.Flush() }()

			if err := cmd.Wait(); err != nil {
				group.Fail(err, "Error")
				return err
			}

			group.Success("Success")
			return nil
		}),
	)), nil
}
