// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ukify

import (
	"bytes"
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
)

var (
	//go:embed Dockerfile
	dockerfile []byte
)

type docker struct {
	client  *client.Client
	image   string
	command string
}

func setupParams(inLinux, inInitrd, inOutput string, inOpts Options) (linux, initrd, output string, opts Options, binds []string, err error) {
	const (
		inDir  = "/in"
		outDir = "/out"
	)

	srcLinux, err := filepath.Abs(inLinux)
	if err != nil {
		return "", "", "", Options{}, nil, err
	}

	linux = filepath.Join(inDir, "linux")
	binds = append(binds, xdocker.Bind(srcLinux, linux, xdocker.ReadOnly))

	srcInitrd, err := filepath.Abs(inInitrd)
	if err != nil {
		return "", "", "", Options{}, nil, err
	}

	initrd = filepath.Join(inDir, "initrd")
	binds = append(binds, xdocker.Bind(srcInitrd, initrd, xdocker.ReadOnly))

	srcOutputDir, err := filepath.Abs(filepath.Dir(inOutput))
	if err != nil {
		return "", "", "", Options{}, nil, err
	}

	output = filepath.Join(outDir, filepath.Base(inOutput))
	binds = append(binds, xdocker.Bind(srcOutputDir, outDir, 0))

	var (
		srcStub = inOpts.Stub
		stub    string
	)
	if srcStub != "" {
		srcStub, err = filepath.Abs(srcStub)
		if err != nil {
			return "", "", "", Options{}, nil, err
		}

		stub = filepath.Join(inDir, "stub")
		binds = append(binds, xdocker.Bind(srcStub, stub, xdocker.ReadOnly))
	}

	opts = Options{
		Stub:    stub,
		Cmdline: inOpts.Cmdline,
	}

	return linux, initrd, output, opts, binds, nil
}

func (d *docker) Run(ctx context.Context, linux, initrd, output string, opts Options) error {
	linux, initrd, output, opts, binds, err := setupParams(linux, initrd, output, opts)
	if err != nil {
		return err
	}

	args := buildArgs(linux, initrd, output, opts)
	log := ulog.FromContext(ctx).With("Command", d.command, "Args", args)

	var (
		out     bytes.Buffer
		spinner = log.Group("ukify")

		stderrGroupWriter = ulog.GroupWriter(spinner)
		stdoutGroupWriter = ulog.GroupWriter(spinner)
	)

	cmd := dexec.Command(ctx, d.client, d.image, d.command, args...)
	cmd.Binds = binds
	cmd.Stderr = io.MultiWriter(stderrGroupWriter, &out)
	cmd.Stdout = io.MultiWriter(stdoutGroupWriter, &out)

	err = cmd.Run()
	_ = stderrGroupWriter.Flush()
	_ = stdoutGroupWriter.Flush()

	if err != nil {
		spinner.Fail(err, "Error")
		return fmt.Errorf("exec: %w (%s)", err, out.String())
	}

	spinner.Success("Success")
	return nil
}

func NewDocker(client *client.Client, image, command string) Executor {
	return &docker{
		client:  client,
		image:   image,
		command: command,
	}
}

func BuildImage(ctx context.Context, client *client.Client) (string, error) {
	return dockerimages.BuildHashed(ctx, client, "ukify", dockerfile)
}

var imgCache = dockerimages.NewClientCache(BuildImage)

func DefaultDocker(ctx context.Context, c *client.Client) (Executor, error) {
	img, err := imgCache.GetOrBuild(ctx, c)
	if err != nil {
		return nil, err
	}

	return NewDocker(c, img, "ukify"), nil
}
