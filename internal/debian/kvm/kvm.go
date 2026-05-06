// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package kvm

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ironcore-dev/os-images/internal/debian"
	"github.com/ironcore-dev/os-images/internal/debian/cloudimages"
	"github.com/ironcore-dev/os-images/internal/debian/kvm/config"
	"github.com/ironcore-dev/os-images/internal/httpfs"
	"github.com/ironcore-dev/os-images/internal/ulog"
	"github.com/ironcore-dev/os-images/internal/xhash"
	"github.com/ironcore-dev/os-images/internal/xio"
	"github.com/ironcore-dev/os-images/internal/xos"
	"github.com/ironcore-dev/os-images/internal/xtemplate"
)

// Builder builds metal Debian images.
type Builder struct {
	repo *cloudimages.Repository
}

func NewBuilder(
	repo *cloudimages.Repository,
) *Builder {
	return &Builder{
		repo: repo,
	}
}

func DefaultBuilder() *Builder {
	return NewBuilder(
		cloudimages.DefaultRepository(),
	)
}

type vars struct {
	Arch          string
	ConsoleDevice string
}

type renderedConfig struct {
	Codename debian.Codename
	Version  cloudimages.Version
	Cmdline  string
}

func (b *Builder) renderConfig(arch string, cfg *config.Config) (*renderedConfig, error) {
	vs := &vars{
		Arch:          arch,
		ConsoleDevice: debian.ConsoleDevice(arch),
	}

	codename, err := debian.ParseCodename(cfg.Codename)
	if err != nil {
		return nil, err
	}

	version, err := cloudimages.ParseVersion(cfg.Version)
	if err != nil {
		return nil, err
	}

	cmdline, err := xtemplate.RenderString(cfg.Cmdline, vs)
	if err != nil {
		return nil, err
	}

	return &renderedConfig{
		Codename: codename,
		Version:  version,
		Cmdline:  cmdline,
	}, nil
}

func (b *Builder) Build(ctx context.Context, arch string, cfg *config.Config, outputDir string) error {
	renderedCfg, err := b.renderConfig(arch, cfg)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	srcFilename := fmt.Sprintf("debian-%s-genericcloud-%s.raw", renderedCfg.Codename.Major(), arch)

	dstFilename := "raw.rootfs"
	if err := b.downloadFile(ctx, outputDir, renderedCfg.Codename, renderedCfg.Version, srcFilename, dstFilename); err != nil {
		return err
	}

	cmdlineFile := filepath.Join(outputDir, "cmdline")
	if err := os.WriteFile(cmdlineFile, []byte(renderedCfg.Cmdline), 0o644); err != nil {
		return fmt.Errorf("writing %s cmdline: %w", arch, err)
	}
	return nil
}

func (b *Builder) checksumAndProgress(ctx context.Context, codename debian.Codename, version cloudimages.Version, filename, msg string, f httpfs.File) (io.ReadCloser, error) {
	log := ulog.FromContext(ctx)

	checksums, err := b.repo.Checksums(ctx, codename, version)
	if err != nil {
		return nil, err
	}

	checksum, err := checksums.Checksum(filename)
	if err != nil {
		return nil, err
	}

	return ulog.ProgressReadCloser(log,
		xhash.ChecksumReadCloser(f, checksums.NewHash(), checksum),
		msg,
	), nil
}

func (b *Builder) openFileForDownload(
	ctx context.Context,
	codename debian.Codename,
	version cloudimages.Version,
	filename string,
) (io.ReadCloser, error) {
	f, err := b.repo.OpenFile(ctx, codename, version, filename)
	if err != nil {
		return nil, fmt.Errorf("opening file %q: %w", filename, err)
	}

	rc, err := b.checksumAndProgress(ctx, codename, version, filename, "Downloading", f)
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("tracking checksum / progress %q: %w", filename, err)
	}

	return xio.ReaderAndCloser(rc, xio.JoinCloser(f, rc)), nil
}

func (b *Builder) downloadFile(
	ctx context.Context,
	outputDir string,
	codename debian.Codename,
	version cloudimages.Version,
	srcFilename, dstFilename string,
) error {
	rc, err := b.openFileForDownload(ctx, codename, version, srcFilename)
	if err != nil {
		return fmt.Errorf("opening file %q for download: %w", srcFilename, err)
	}
	defer func() { _ = rc.Close() }()

	if err := xos.WriteFileReader(filepath.Join(outputDir, dstFilename), rc, 0644); err != nil {
		return fmt.Errorf("writing file %q: %w", dstFilename, err)
	}

	return nil
}
