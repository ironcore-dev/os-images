// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sqfstar

import (
	"context"
	"io"
	osexec "os/exec"

	"github.com/docker/docker/client"
)

type Executor interface {
	Run(ctx context.Context, rd io.Reader) (io.ReadCloser, error)
}

func GetExecutor(
	ctx context.Context,
	getClient func(ctx context.Context) (*client.Client, error),
	getBaseTempDir func(ctx context.Context) (string, error),
) (Executor, error) {
	if _, err := osexec.LookPath("sqfstar"); err == nil {
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
