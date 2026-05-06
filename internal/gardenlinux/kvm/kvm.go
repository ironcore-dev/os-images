package kvm

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ironcore-dev/os-images/internal/gardenlinux/releases"
	"github.com/ironcore-dev/os-images/internal/ulog"
	"github.com/ironcore-dev/os-images/internal/xarchives"
	"github.com/ironcore-dev/os-images/internal/xtemplate"
	"github.com/mholt/archives"
)

type Builder struct {
	releases *releases.Releases
}

func NewBuilder(releases *releases.Releases) *Builder {
	return &Builder{releases: releases}
}

func DefaultBuilder() *Builder {
	return NewBuilder(releases.DefaultReleases())
}

type Config struct {
	Tag     string
	Flavor  string
	Cmdline string
}

type renderedConfig struct {
	Arch    string
	Tag     string
	Flavor  string
	Cmdline string
}

type vars struct {
	Arch          string
	ConsoleDevice string
}

func consoleDevice(arch string) string {
	if arch == "arm64" {
		return "ttyAMA0"
	}
	return "ttyS0"
}

func (b *Builder) renderConfig(arch string, cfg *Config) (*renderedConfig, error) {
	vs := &vars{
		Arch:          arch,
		ConsoleDevice: consoleDevice(arch),
	}

	cmdline, err := xtemplate.RenderString(cfg.Cmdline, vs)
	if err != nil {
		return nil, err
	}

	return &renderedConfig{
		Arch:    arch,
		Tag:     cfg.Tag,
		Flavor:  cfg.Flavor,
		Cmdline: cmdline,
	}, nil
}

func (b *Builder) Build(ctx context.Context, arch string, cfg *Config, outputDir string) error {
	log := ulog.FromContext(ctx)

	renderedCfg, err := b.renderConfig(arch, cfg)
	if err != nil {
		return err
	}

	asset, err := b.releases.GetAsset(ctx, renderedCfg.Flavor, renderedCfg.Arch, renderedCfg.Tag)
	if err != nil {
		return err
	}

	rc, err := asset.Open(ctx)
	if err != nil {
		return err
	}

	rc = ulog.ProgressReadCloser(log, rc, "Downloading asset", "Asset", asset.Name())
	defer func() { _ = rc.Close() }()

	if err := extractArchive(ctx, asset.Base(), rc, outputDir); err != nil {
		return err
	}

	cmdlineFilename := filepath.Join(outputDir, "cmdline")
	if err := os.WriteFile(cmdlineFilename, []byte(renderedCfg.Cmdline), 0o644); err != nil {
		return fmt.Errorf("writing %s cmdline: %w", arch, err)
	}
	return nil
}

func extractArchive(ctx context.Context, assetBase string, rd io.Reader, outputDir string) error {
	pickFiles := xarchives.PickFiles(
		xarchives.PickFileIf("rootfs.raw",
			xarchives.FileNameEquals(fmt.Sprintf("%s.raw", assetBase)),
			xarchives.WriteTo(filepath.Join(outputDir, "rootfs.raw")),
		),
	)

	err := archives.CompressedArchive{
		Extraction:  archives.Tar{},
		Compression: archives.Xz{},
	}.Extract(ctx, rd, pickFiles.Handle)
	if err != nil {
		return fmt.Errorf("extracting archive: %w", err)
	}

	if err := pickFiles.Validate(); err != nil {
		return err
	}
	return nil
}
