// Copyright 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package metal

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cavaliergopher/cpio"
	"github.com/ironcore-dev/os-images/internal/debian"
	"github.com/ironcore-dev/os-images/internal/debian/metal/config"
	"github.com/ironcore-dev/os-images/internal/tools/mmdebstrap"
	"github.com/ironcore-dev/os-images/internal/tools/sqfstar"
	"github.com/ironcore-dev/os-images/internal/tools/ukify"
	"github.com/ironcore-dev/os-images/internal/ulog"
	"github.com/ironcore-dev/os-images/internal/xos"
	"github.com/ironcore-dev/os-images/internal/xtemplate"
	"github.com/klauspost/compress/zstd"
	"github.com/mholt/archives"
)

var (
	//go:embed assets/scripts_squashfs.sh
	bootScript []byte

	//go:embed assets/hooks_squashfs.sh
	hookScript []byte

	//go:embed assets/ignition_uuid_fetch.sh
	ignitionUUIDFetchScript []byte

	//go:embed assets/ignition_uuid_fetch.service
	ignitionUUIDFetchService []byte
)

type renderedConfig struct {
	Arch       string
	Suite      string
	Variant    string
	Packages   []string
	Cmdline    string
	Components []string
}

// Builder builds metal Debian images.
type Builder struct {
	mmdebstrap mmdebstrap.Executor
	sqfstar    sqfstar.Executor
	ukify      ukify.Executor
}

func NewBuilder(
	mmdebstrap mmdebstrap.Executor,
	sqfstar sqfstar.Executor,
	ukify ukify.Executor,
) *Builder {
	return &Builder{
		mmdebstrap: mmdebstrap,
		sqfstar:    sqfstar,
		ukify:      ukify,
	}
}

func DefaultBuilder(
	mmdebstrap mmdebstrap.Executor,
	sqfstar sqfstar.Executor,
	ukify ukify.Executor,
) *Builder {
	return NewBuilder(
		mmdebstrap,
		sqfstar,
		ukify,
	)
}

type vars struct {
	Arch          string
	Suite         string
	ConsoleDevice string
}

func (b *Builder) renderConfig(arch string, cfg *config.Config) (*renderedConfig, error) {
	vs := &vars{
		Arch:          arch,
		Suite:         cfg.Suite,
		ConsoleDevice: debian.ConsoleDevice(arch),
	}

	packages := make([]string, 0, len(cfg.Packages))
	for _, cfgPkg := range cfg.Packages {
		pkg, err := xtemplate.RenderString(cfgPkg, vs)
		if err != nil {
			return nil, err
		}

		packages = append(packages, pkg)
	}

	cmdline, err := xtemplate.RenderString(cfg.Cmdline, vs)
	if err != nil {
		return nil, err
	}

	return &renderedConfig{
		Arch:       arch,
		Suite:      cfg.Suite,
		Variant:    cfg.Variant,
		Packages:   packages,
		Cmdline:    cmdline,
		Components: cfg.Components,
	}, nil
}

func (b *Builder) Build(ctx context.Context, arch string, cfg *config.Config, outputDir string) error {
	renderedCfg, err := b.renderConfig(arch, cfg)
	if err != nil {
		return err
	}

	tarPath := filepath.Join(outputDir, "rootfs.tar")
	defer func() { _ = os.Remove(tarPath) }()

	if err := b.debootstrap(ctx, renderedCfg, tarPath); err != nil {
		return fmt.Errorf("[%s] building rootfs: %w", arch, err)
	}

	if err := extractBootArtifacts(tarPath, outputDir, arch); err != nil {
		return fmt.Errorf("[%s] extracting boot-artifacts: %w", arch, err)
	}

	if err := b.buildSquashfs(ctx, tarPath, filepath.Join(outputDir, "root.squashfs")); err != nil {
		return fmt.Errorf("[%s] building squashfs: %w", arch, err)
	}

	if err := b.buildUKI(ctx, outputDir, arch, renderedCfg); err != nil {
		return fmt.Errorf("[%s] building UKI: %w", arch, err)
	}

	return nil
}

func essentialHooks() []mmdebstrap.Hook {
	return []mmdebstrap.Hook{
		// The target directories are created by initramfs-tools, which is not
		// yet installed at the essential-hook stage. Create them explicitly so
		// the copy-in hooks that follow can succeed.
		{Hook: `mkdir -p "$1/usr/share/initramfs-tools/scripts" "$1/usr/share/initramfs-tools/hooks" "$1/usr/local/bin" "$1/etc/systemd/system" "$1/etc/systemd/system/ignition-fetch.service.wants"`},
		{
			CopyIn: &mmdebstrap.CopyIn{
				SrcContent:  bootScript,
				Mode:        0o755,
				DstFilename: "/usr/share/initramfs-tools/scripts/squashfs",
			},
		},
		{
			CopyIn: &mmdebstrap.CopyIn{
				SrcContent:  hookScript,
				Mode:        0o755,
				DstFilename: "/usr/share/initramfs-tools/hooks/squashfs",
			},
		},
		// Ignition UUID-append: fetch config with machine UUID appended to URL.
		{
			CopyIn: &mmdebstrap.CopyIn{
				SrcContent:  ignitionUUIDFetchScript,
				Mode:        0o755,
				DstFilename: "/usr/local/bin/ignition-uuid-fetch",
			},
		},
		{
			CopyIn: &mmdebstrap.CopyIn{
				SrcContent:  ignitionUUIDFetchService,
				Mode:        0o644,
				DstFilename: "/etc/systemd/system/ignition-uuid-fetch.service",
			},
		},
		// Enable the unit so it runs before ignition-fetch.service.
		{Hook: `ln -s /etc/systemd/system/ignition-uuid-fetch.service "$1/etc/systemd/system/ignition-fetch.service.wants/ignition-uuid-fetch.service"`},
	}
}

func (b *Builder) debootstrap(ctx context.Context, cfg *renderedConfig, tarPath string) error {
	rc, err := b.mmdebstrap.Run(ctx, cfg.Suite, mmdebstrap.Options{
		Variant:        cfg.Variant,
		Architectures:  []string{cfg.Arch},
		Components:     cfg.Components,
		Include:        cfg.Packages,
		EssentialHooks: essentialHooks(),
	})
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()

	if err := xos.WriteFileReader(tarPath, rc, 0755); err != nil {
		return fmt.Errorf("writing tar: %w", err)
	}
	return nil
}

func efiStubName(arch string) string {
	if arch == "arm64" {
		return "linuxaa64.efi.stub"
	}
	return "linuxx64.efi.stub"
}

// extractBootArtifacts extracts vmlinuz, initrd, and EFI stub from a rootfs tar.
// Only these specific files are needed on disk; the rest stays in the tar for sqfstar.
func extractBootArtifacts(tarPath, outputDir, arch string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	stubName := efiStubName(arch)
	stubSuffix := filepath.Join("usr", "lib", "systemd", "boot", "efi", stubName)

	var vmlinuz, initrd, stub string

	err = archives.Tar{}.Extract(context.Background(), f, func(_ context.Context, info archives.FileInfo) error {
		name := filepath.Clean(info.NameInArchive)
		base := filepath.Base(name)

		var dstName string
		switch {
		case strings.HasPrefix(base, "vmlinuz-") && strings.HasPrefix(name, "boot/"):
			dstName = "vmlinuz"
			vmlinuz = dstName
		case strings.HasPrefix(base, "initrd.img-") && strings.HasPrefix(name, "boot/"):
			dstName = "initrd"
			initrd = dstName
		case strings.HasSuffix(name, stubSuffix):
			dstName = stubName
			stub = dstName
		default:
			return nil
		}

		src, err := info.Open()
		if err != nil {
			return err
		}
		defer func() { _ = src.Close() }()

		return xos.WriteFileReader(filepath.Join(outputDir, dstName), src, 0o644)
	})
	if err != nil {
		return err
	}

	if vmlinuz == "" {
		return fmt.Errorf("vmlinuz not found in tar")
	}
	if initrd == "" {
		return fmt.Errorf("initrd not found in tar")
	}
	if stub == "" {
		return fmt.Errorf("EFI stub %s not found in tar", stubName)
	}
	return nil
}

func (b *Builder) buildSquashfs(ctx context.Context, tarPath, outputPath string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("opening tar: %w", err)
	}
	defer func() { _ = f.Close() }()

	rc, err := b.sqfstar.Run(ctx, f)
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()

	if err := xos.WriteFileReader(outputPath, rc, 0o644); err != nil {
		return fmt.Errorf("writing squashfs: %w", err)
	}
	return nil
}

func (b *Builder) buildUKI(ctx context.Context, archDir, arch string, renderedCfg *renderedConfig) error {
	log := ulog.FromContext(ctx)

	initrd := filepath.Join(archDir, "initrd")
	squashfsPath := filepath.Join(archDir, "root.squashfs")
	initrdUKI := filepath.Join(archDir, "initrd-uki")

	group := log.Group("Append squashfs")
	if err := xos.CopyFile(initrd, initrdUKI); err != nil {
		return fmt.Errorf("copying initrd for UKI: %w", err)
	}
	if err := appendSquashfsCPIO(squashfsPath, initrdUKI); err != nil {
		group.Fail(err, "Error appending squashfs")
		return fmt.Errorf("appending squashfs to initrd: %w", err)
	}
	group.Success("Appended squashfs")

	stubPath := filepath.Join(archDir, efiStubName(arch))
	if _, err := os.Stat(stubPath); err != nil {
		return fmt.Errorf("EFI stub not found at %s: %w", stubPath, err)
	}

	if err := b.ukify.Run(ctx,
		filepath.Join(archDir, "vmlinuz"),
		initrdUKI,
		filepath.Join(archDir, "uki.img"),
		ukify.Options{
			Stub:    stubPath,
			Cmdline: renderedCfg.Cmdline,
		},
	); err != nil {
		return fmt.Errorf("ukify: %w", err)
	}

	_ = os.Remove(initrdUKI)
	_ = os.Remove(stubPath)
	return nil
}

// appendSquashfsCPIO appends the squashfs file to the initrd as a cpio entry,
// compressed with zstd. Equivalent to:
//
//	echo root.squashfs | cpio -H newc -o | zstd >> initrd-uki
func appendSquashfsCPIO(squashfsPath, initrdPath string) error {
	sqf, err := os.Open(squashfsPath)
	if err != nil {
		return fmt.Errorf("opening squashfs: %w", err)
	}
	defer func() { _ = sqf.Close() }()

	sqInfo, err := sqf.Stat()
	if err != nil {
		return fmt.Errorf("stat squashfs: %w", err)
	}

	out, err := os.OpenFile(initrdPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	zw, err := zstd.NewWriter(out)
	if err != nil {
		return fmt.Errorf("creating zstd writer: %w", err)
	}

	cw := cpio.NewWriter(zw)
	defer func() { _ = cw.Close() }()

	if err := cw.WriteHeader(&cpio.Header{
		Name: "root.squashfs",
		Mode: 0o100644,
		Size: sqInfo.Size(),
	}); err != nil {
		return fmt.Errorf("writing cpio header: %w", err)
	}

	if _, err := io.Copy(cw, sqf); err != nil {
		return fmt.Errorf("writing cpio data: %w", err)
	}

	if err := cw.Close(); err != nil {
		_ = os.Remove(out.Name())
		return fmt.Errorf("closing cpio archive: %w", err)
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("closing zstd writer: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("closing output file: %w", err)
	}

	return nil
}
