// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ukify

import (
	"bytes"
	"context"
	"fmt"
	"io"
	osexec "os/exec"

	"github.com/ironcore-dev/os-images/internal/ulog"
)

type exec struct {
	command string
}

func NewExec(command string) Executor {
	return &exec{command}
}

func DefaultExec() Executor {
	return NewExec("ukify")
}

func (e *exec) Run(ctx context.Context, linux, initrd, output string, opts Options) error {
	args := buildArgs(linux, initrd, output, opts)
	log := ulog.FromContext(ctx)

	var (
		out   bytes.Buffer
		group = log.Group("ukify", "Command", e.command, "Args", args)

		stderrGroupWriter = ulog.GroupWriter(group)
		stdoutGroupWriter = ulog.GroupWriter(group)
	)

	cmd := osexec.CommandContext(ctx, e.command, args...)
	cmd.Stderr = io.MultiWriter(stderrGroupWriter, &out)
	cmd.Stdout = io.MultiWriter(stdoutGroupWriter, &out)

	err := cmd.Run()
	_ = stderrGroupWriter.Flush()
	_ = stdoutGroupWriter.Flush()

	if err != nil {
		group.Fail(err, "Error")
		return fmt.Errorf("exec: %w (%s)", err, out.String())
	}

	group.Success("Success")
	return nil
}
