// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package dexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type Cmd struct {
	ctx context.Context

	mu         sync.Mutex
	started    bool
	done       chan struct{}
	res        error
	statusCode int

	pipes []io.Closer

	client *client.Client

	Image string
	Path  string
	Args  []string

	Binds []string

	Stdin  io.Reader
	Stderr io.Writer
	Stdout io.Writer
}

func Command(ctx context.Context, c *client.Client, image string, name string, args ...string) *Cmd {
	return &Cmd{
		ctx:    ctx,
		client: c,
		Image:  image,
		Path:   name,
		Args:   append([]string{name}, args...),
	}
}

func (c *Cmd) StdoutPipe() (io.ReadCloser, error) {
	if c.Stdout != nil {
		return nil, errors.New("stdout already set")
	}

	pr, pw := io.Pipe()
	c.Stdout = pw
	c.pipes = append(c.pipes, pw)
	return pr, nil
}

func (c *Cmd) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return errors.New("already started")
	}

	res, err := c.client.ContainerCreate(c.ctx,
		&container.Config{
			Image: c.Image,
			Cmd:   c.Args,
		},
		&container.HostConfig{
			Binds: c.Binds,
		},
		nil,
		nil,
		"",
	)
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}

	var (
		needStdin  = c.Stdin != nil
		needStdout = c.Stdout != nil
		needStderr = c.Stderr != nil
		needAttach = needStdin || needStdout || needStderr
		attach     types.HijackedResponse
	)
	if needAttach {
		newAttached, err := c.client.ContainerAttach(c.ctx, res.ID, container.AttachOptions{
			Stream: true,
			Stdin:  needStdin,
			Stdout: needStdout,
			Stderr: needStderr,
		})
		if err != nil {
			return fmt.Errorf("attach container: %w", err)
		}

		attach = newAttached
	}

	var wg sync.WaitGroup
	if needStdin {
		wg.Go(func() {
			defer func() { _ = attach.CloseWrite() }()

			_, _ = io.Copy(attach.Conn, c.Stdin)
		})
	}

	if needStdout || needStderr {
		wg.Go(func() {
			var (
				stdout = io.Discard
				stderr = io.Discard
			)
			if needStdout {
				stdout = c.Stdout
			}
			if needStderr {
				stderr = c.Stderr
			}
			_, _ = stdcopy.StdCopy(stdout, stderr, attach.Reader)
		})
	}

	statusChan, errChan := c.client.ContainerWait(c.ctx, res.ID, container.WaitConditionNextExit)

	wg.Go(func() {
		select {
		case err := <-errChan:
			c.res = err
		case status := <-statusChan:
			if status.Error != nil {
				c.res = errors.New(status.Error.Message)
			}
			c.statusCode = int(status.StatusCode)
		}
	})

	c.started = true
	c.done = make(chan struct{})

	if err := c.client.ContainerStart(c.ctx, res.ID, container.StartOptions{}); err != nil {
		if attach.Conn != nil {
			attach.Close()
		}
		_ = c.client.ContainerRemove(c.ctx, res.ID, container.RemoveOptions{Force: true})
		for _, pipe := range c.pipes {
			_ = pipe.Close()
		}
		close(c.done)
		return fmt.Errorf("start container: %w", err)
	}

	go func() {
		defer close(c.done)
		defer func() {
			_ = c.client.ContainerRemove(c.ctx, res.ID, container.RemoveOptions{
				Force: true,
			})
		}()
		defer func() {
			for _, pipe := range c.pipes {
				_ = pipe.Close()
			}
		}()

		wg.Wait()
	}()

	return nil
}

func (c *Cmd) Wait() error {
	<-c.done
	return c.res
}

func (c *Cmd) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

func (c *Cmd) CombinedOutput() ([]byte, error) {
	var out bytes.Buffer
	c.Stdout = &out
	c.Stderr = &out
	err := c.Run()
	return out.Bytes(), err
}
