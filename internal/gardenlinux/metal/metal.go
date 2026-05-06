package metal

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/ironcore-dev/os-images/internal/gardenlinux/releases"
	"github.com/ironcore-dev/os-images/internal/tools/ukify"
	"github.com/ironcore-dev/os-images/internal/ulog"
	"github.com/ironcore-dev/os-images/internal/xarchives"
	"github.com/ironcore-dev/os-images/internal/xtemplate"
	"github.com/mholt/archives"
)

type Builder struct {
	releases *releases.Releases
	ukify    ukify.Executor
}

type Bar struct {
	Baz string
}

func NewBuilder(
	releases *releases.Releases,
	ukify ukify.Executor,
) *Builder {
	return &Builder{
		releases: releases,
		ukify:    ukify,
	}
}

func DefaultBuilder(ukify ukify.Executor) *Builder {
	return NewBuilder(
		releases.DefaultReleases(),
		ukify,
	)
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

	if err := extractArchive(ctx, renderedCfg.Arch, asset.Base(), rc, outputDir); err != nil {
		return err
	}

	if err := b.ukify.Run(ctx,
		filepath.Join(outputDir, "vmlinuz"),
		filepath.Join(outputDir, "initrd"),
		filepath.Join(outputDir, "uki.img"),
		ukify.Options{
			Stub:    filepath.Join(outputDir, "stub"),
			Cmdline: renderedCfg.Cmdline,
		},
	); err != nil {
		return fmt.Errorf("ukify: %w", err)
	}
	return nil
}

func extractArchive(ctx context.Context, arch, assetBase string, rd io.Reader, outputDir string) error {
	pickFiles := xarchives.PickFiles(
		xarchives.PickFileIf("pxe-artifacts",
			xarchives.FileNameEquals(fmt.Sprintf("%s.pxe.tar.gz", assetBase)),
			func(ctx context.Context, info archives.FileInfo) error {
				f, err := info.Open()
				if err != nil {
					return err
				}
				defer func() { _ = f.Close() }()

				pxeFiles := xarchives.PickFiles(
					xarchives.PickFileIf("initrd",
						xarchives.FileNameEquals("initrd"),
						xarchives.WriteTo(filepath.Join(outputDir, "initrd")),
					),
					xarchives.PickFileIf("vmlinuz",
						xarchives.FileNameEquals("vmlinuz"),
						xarchives.WriteTo(filepath.Join(outputDir, "vmlinuz")),
					),
					xarchives.PickFileIf("root.squashfs",
						xarchives.FileNameEquals("root.squashfs"),
						xarchives.WriteTo(filepath.Join(outputDir, "root.squashfs")),
					),
				)

				if err := (archives.CompressedArchive{
					Extraction:  archives.Tar{},
					Compression: archives.Gz{},
				}).Extract(ctx, f, pxeFiles.Handle); err != nil {
					return err
				}
				return pxeFiles.Validate()
			},
		),
		xarchives.PickFileIf("base-tar",
			xarchives.FileNameEquals(fmt.Sprintf("%s.tar", assetBase)),
			func(ctx context.Context, info archives.FileInfo) error {
				f, err := info.Open()
				if err != nil {
					return err
				}
				defer func() { _ = f.Close() }()

				var efiStubName string
				switch arch {
				case "amd64":
					efiStubName = "linuxx64.efi.stub"
				case "arm64":
					efiStubName = "linuxaa64.efi.stub"
				default:
					return fmt.Errorf("could not determine efi stub for arch %s", arch)
				}

				pickFiles := xarchives.PickFiles(
					xarchives.PickFileIf("stub",
						xarchives.FileNameEquals(efiStubName),
						xarchives.WriteTo(filepath.Join(outputDir, "stub")),
					),
				)

				if err := (archives.Tar{}).Extract(ctx, f, pickFiles.Handle); err != nil {
					return err
				}

				return pickFiles.Validate()
			},
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
