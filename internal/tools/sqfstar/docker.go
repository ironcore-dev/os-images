// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package sqfstar

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
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

func (d *docker) Run(ctx context.Context, rd io.Reader) (io.ReadCloser, error) {
	log := ulog.FromContext(ctx)

	tmpFile, err := os.CreateTemp(d.tempDir, "sqfstar-*.squashfs")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	// sqfstar needs to create the file itself; close and remove the placeholder.
	_ = tmpFile.Close()
	_ = os.Remove(tmpPath)

	outDir, err := filepath.Abs(filepath.Dir(tmpPath))
	if err != nil {
		return nil, fmt.Errorf("abs of temp dir: %w", err)
	}

	const containerOutDir = "/out"
	containerOutput := filepath.Join(containerOutDir, filepath.Base(tmpPath))

	args := buildArgs(containerOutput)

	var out bytes.Buffer
	group := log.Group("sqfstar")
	groupWriter := ulog.GroupWriter(group)

	cmd := dexec.Command(ctx, d.client, d.image, d.command, args...)
	cmd.Binds = []string{xdocker.Bind(outDir, containerOutDir, 0)}
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
	return dockerimages.BuildHashed(ctx, client, "sqfstar", dockerfile)
}

var imgCache = dockerimages.NewClientCache(BuildImage)

func DefaultDocker(ctx context.Context, c *client.Client, tempDir string) (Executor, error) {
	img, err := imgCache.GetOrBuild(ctx, c)
	if err != nil {
		return nil, err
	}

	return NewDocker(c, tempDir, img, "sqfstar"), nil
}
