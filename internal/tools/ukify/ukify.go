// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ukify

import (
	"context"
	osexec "os/exec"

	"github.com/docker/docker/client"
)

type Options struct {
	Stub    string
	Cmdline string
}

type Executor interface {
	Run(ctx context.Context, linux, initrd, output string, opts Options) error
}

func GetExecutor(ctx context.Context, getClient func(ctx context.Context) (*client.Client, error)) (Executor, error) {
	if _, err := osexec.LookPath("ukify"); err == nil {
		return NewExec("ukify"), nil
	}

	c, err := getClient(ctx)
	if err != nil {
		return nil, err
	}

	return DefaultDocker(ctx, c)
}
