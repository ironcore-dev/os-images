// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package xarchives

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ironcore-dev/os-images/internal/xos"
	"github.com/mholt/archives"
)

func ExtractToDir(dir string) archives.FileHandler {
	return func(ctx context.Context, info archives.FileInfo) error {
		filename := filepath.Join(dir, info.NameInArchive)
		if info.IsDir() {
			return os.MkdirAll(filename, info.Mode())
		}

		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			return err
		}

		// Handle symlinks.
		if info.LinkTarget != "" && info.Mode()&os.ModeSymlink != 0 {
			_ = os.Remove(filename) // tar entries may overwrite earlier ones
			return os.Symlink(info.LinkTarget, filename)
		}

		// Handle hard links.
		if info.LinkTarget != "" {
			_ = os.Remove(filename)
			return os.Link(filepath.Join(dir, info.LinkTarget), filename)
		}

		f, err := info.Open()
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()

		return xos.WriteFileReader(filename, f, info.Mode())
	}
}

type FilePicker interface {
	Name() string
	Handle(ctx context.Context, info archives.FileInfo) (bool, error)
}

type filterFilePicker struct {
	name      string
	predicate func(info fs.FileInfo) bool
	action    func(ctx context.Context, info archives.FileInfo) error
}

func (f *filterFilePicker) Name() string {
	return f.name
}

func (f *filterFilePicker) Handle(ctx context.Context, info archives.FileInfo) (bool, error) {
	if !f.predicate(info) {
		return false, nil
	}
	if err := f.action(ctx, info); err != nil {
		return false, err
	}
	return true, nil
}

func PickFileIf(
	name string,
	predicate func(info fs.FileInfo) bool,
	action func(ctx context.Context, info archives.FileInfo) error,
) FilePicker {
	return &filterFilePicker{
		name:      name,
		predicate: predicate,
		action:    action,
	}
}

func FileNameEquals(name string) func(info fs.FileInfo) bool {
	return func(info fs.FileInfo) bool {
		return info.Name() == name
	}
}

func WriteTo(filename string) func(ctx context.Context, info archives.FileInfo) error {
	return func(ctx context.Context, info archives.FileInfo) error {
		f, err := info.Open()
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()

		return xos.WriteFileReader(filename, f, info.Mode())
	}
}

type RequiredFilesHandler struct {
	required []FilePicker
}

func PickFiles(required ...FilePicker) *RequiredFilesHandler {
	return &RequiredFilesHandler{
		required: required,
	}
}

func (p *RequiredFilesHandler) Validate() error {
	if len(p.required) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("missing files: ")
	for i, missing := range p.required {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(missing.Name())
	}
	return errors.New(sb.String())
}

func (p *RequiredFilesHandler) Handle(ctx context.Context, info archives.FileInfo) error {
	if len(p.required) == 0 {
		return fs.SkipAll
	}

	for i, pick := range p.required {
		ok, err := pick.Handle(ctx, info)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}

		p.required = slices.Delete(p.required, i, i+1)
		if len(p.required) == 0 {
			return fs.SkipAll
		}
		return nil
	}
	return nil
}
