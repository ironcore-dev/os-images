// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"context"
	"errors"
	"os"
	"sync"

	"github.com/docker/docker/client"
)

type provider struct {
	mu sync.RWMutex

	baseTempDirParentFlag *string

	clientRes      *result[*client.Client]
	baseTempDirRes *result[string]
}

func NewProvider(baseTempDirParentFlag *string) Provider {
	return &provider{
		baseTempDirParentFlag: baseTempDirParentFlag,
	}
}

type result[E any] struct {
	value E
	err   error
}

func newResult[E any](e E, err error) *result[E] {
	return &result[E]{value: e, err: err}
}

func (r *result[E]) ForEach(f func(E)) {
	if r == nil {
		return
	}
	if r.err == nil {
		f(r.value)
	}
}

func (r *result[E]) Values() (E, error) {
	return r.value, r.err
}

func (p *provider) GetClient(ctx context.Context) (*client.Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.clientRes != nil {
		return p.clientRes.Values()
	}

	p.clientRes = newResult(client.NewClientWithOpts(client.FromEnv))
	return p.clientRes.Values()
}

func (p *provider) getOrCreateBaseTempDir() (string, error) {
	baseTempDir := *p.baseTempDirParentFlag
	if baseTempDir == "" {
		return "", nil
	}

	return os.MkdirTemp(baseTempDir, ".os-images-tmp-*")
}

func (p *provider) GetBaseTempDir(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.baseTempDirRes != nil {
		return p.baseTempDirRes.Values()
	}

	p.baseTempDirRes = newResult(p.getOrCreateBaseTempDir())
	return p.baseTempDirRes.Values()
}

func (p *provider) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error

	p.clientRes.ForEach(func(client *client.Client) {
		if err := client.Close(); err != nil {
			errs = append(errs, err)
		}
	})

	p.baseTempDirRes.ForEach(func(baseTempDir string) {
		if baseTempDir != "" {
			if err := os.RemoveAll(baseTempDir); err != nil {
				errs = append(errs, err)
			}
		}
	})

	return errors.Join(errs...)
}

type Provider interface {
	GetClient(ctx context.Context) (*client.Client, error)
	GetBaseTempDir(ctx context.Context) (string, error)
	Close() error
}
