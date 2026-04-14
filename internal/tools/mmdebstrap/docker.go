// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package mmdebstrap

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"path/filepath"

	"github.com/docker/docker/client"
	"github.com/ironcore-dev/os-images/internal/dexec"
	dockerimages "github.com/ironcore-dev/os-images/internal/docker-images"
	"github.com/ironcore-dev/os-images/internal/ulog"
	"github.com/ironcore-dev/os-images/internal/xdocker"
	"github.com/ironcore-dev/os-images/internal/xio"
)

var (
	//go:embed Dockerfile
	dockerfile []byte
)

type docker struct {
	client  *client.Client
	tempDir string

	image   string
	command string
}

func setupParams(inSuite string, inOpts Options) (suite string, opts Options, binds []string, err error) {
	suite = inSuite

	const inDir = "/in"

	essentialHooks := make([]Hook, 0, len(inOpts.EssentialHooks))
	for _, inHook := range inOpts.EssentialHooks {
		switch {
		case inHook.Hook != "":
			essentialHooks = append(essentialHooks, inHook)
		case inHook.CopyIn != nil:
			copyIn := inHook.CopyIn
			if copyIn.SrcFilename == "" {
				return "", Options{}, nil, fmt.Errorf("copy-in requires a src-filename")
			}

			inSrcFilename, err := filepath.Abs(copyIn.SrcFilename)
			if err != nil {
				return "", Options{}, nil, fmt.Errorf("abs of %q: %w", copyIn.SrcFilename, err)
			}

			srcFilename := filepath.Join(inDir, inSrcFilename)

			essentialHooks = append(essentialHooks, Hook{
				CopyIn: &CopyIn{
					SrcFilename: srcFilename,
					DstFilename: copyIn.DstFilename,
				},
			})
			binds = append(binds, xdocker.Bind(inSrcFilename, srcFilename, xdocker.ReadOnly))
		}
	}

	opts = Options{
		Variant:        inOpts.Variant,
		Architectures:  inOpts.Architectures,
		Components:     inOpts.Components,
		Include:        inOpts.Include,
		EssentialHooks: essentialHooks,
	}

	return suite, opts, binds, nil
}

func (d *docker) Run(ctx context.Context, suite string, opts Options) (io.ReadCloser, error) {
	cleanupHookTempFiles, err := createHookTempFiles(&opts, d.tempDir)
	if err != nil {
		return nil, err
	}

	suite, opts, binds, err := setupParams(suite, opts)
	if err != nil {
		return nil, err
	}

	args := buildArgs(suite, opts)

	log := ulog.FromContext(ctx)

	cmd := dexec.Command(ctx, d.client, d.image, d.command, args...)
	cmd.Binds = binds
	rc, err := cmd.StdoutPipe()
	if err != nil {
		_ = cleanupHookTempFiles()
		return nil, err
	}

	group := log.Group("mmdebstrap")
	lineWriter := ulog.GroupWriter(group)

	cmd.Stderr = lineWriter
	if err := cmd.Start(); err != nil {
		_ = cleanupHookTempFiles()
		group.Fail(err, "Error")
		return nil, err
	}

	return xio.ReaderAndCloser(rc, xio.JoinCloser(
		rc,
		xio.CloserFunc(func() error {
			defer func() { _ = cleanupHookTempFiles() }()
			defer func() { _ = lineWriter.Flush() }()

			if err := cmd.Wait(); err != nil {
				group.Fail(err, "Error")
				return err
			}

			group.Success("Success")
			return nil
		}),
	)), nil
}

func NewDocker(
	client *client.Client,
	tempDir string,
	image, command string,
) Executor {
	return &docker{
		client:  client,
		tempDir: tempDir,
		image:   image,
		command: command,
	}
}

func BuildImage(ctx context.Context, client *client.Client) (string, error) {
	return dockerimages.BuildHashed(ctx, client, "mmdebstrap", dockerfile)
}

var imgCache = dockerimages.NewClientCache(BuildImage)

func DefaultDocker(ctx context.Context, c *client.Client, tempDir string) (Executor, error) {
	img, err := imgCache.GetOrBuild(ctx, c)
	if err != nil {
		return nil, err
	}

	return NewDocker(c, tempDir, img, "mmdebstrap"), nil
}
