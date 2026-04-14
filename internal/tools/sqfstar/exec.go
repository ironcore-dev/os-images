// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sqfstar

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	osexec "os/exec"

	"github.com/ironcore-dev/os-images/internal/ulog"
	"github.com/ironcore-dev/os-images/internal/xio"
)

type exec struct {
	command string
}

func NewExec(command string) Executor {
	return &exec{command}
}

func DefaultExec() Executor {
	return NewExec("sqfstar")
}

func (e *exec) Run(ctx context.Context, rd io.Reader) (io.ReadCloser, error) {
	log := ulog.FromContext(ctx)

	tmpFile, err := os.CreateTemp("", "sqfstar-*.squashfs")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	// sqfstar needs to create the file itself; close and remove the placeholder.
	_ = tmpFile.Close()
	_ = os.Remove(tmpPath)

	args := buildArgs(tmpPath)

	var out bytes.Buffer
	group := log.Group("sqfstar")
	groupWriter := ulog.GroupWriter(group)

	cmd := osexec.CommandContext(ctx, e.command, args...)
	cmd.Stdin = rd
	cmd.Stdout = io.MultiWriter(groupWriter, &out)
	cmd.Stderr = io.MultiWriter(groupWriter, &out)

	if err := cmd.Run(); err != nil {
		_ = groupWriter.Flush()
		_ = os.Remove(tmpPath)
		group.Fail(err, "Error")
		return nil, fmt.Errorf("sqfstar: %w (%s)", err, out.String())
	}
	_ = groupWriter.Flush()
	group.Success("Success")

	f, err := os.Open(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("opening squashfs output: %w", err)
	}

	return xio.ReaderAndCloser(f, xio.JoinCloser(
		f,
		xio.CloserFunc(func() error {
			return os.Remove(tmpPath)
		}),
	)), nil
}
