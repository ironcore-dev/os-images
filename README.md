# os-images

[![REUSE status](https://api.reuse.software/badge/github.com/ironcore-dev/os-images)](https://api.reuse.software/info/github.com/ironcore-dev/os-images)
[![GitHub License](https://img.shields.io/static/v1?label=License&message=Apache-2.0&color=blue)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://makeapullrequest.com)

## Overview
The `os-images` repository specializes in automating the release and publication of Operating System (OS) image artifacts as OCI (Open Container Initiative) images. This project streamlines the process of pushing these OS images to the project's GitHub Container Registry (`ghcr.io`) via GitHub Actions.

### Published Images

> [!WARNING]
>
> We are in the process of re-structuring the images we publish. The pattern
> described is the desired state which may not be fully implemented yet. Images
> not following this pattern will eventually be deleted, if you require some of
> them to be retained, please open an issue.

Images in this repository are published according to the following scheme:

```
ghcr.io/ironcore-dev/os-images/$distro[/$flavor]:$version[-$variant-$arch][-manual]
```

* `$distro`: The linux distribution from which this image is built. Currently,
  only [gardenlinux](https://gardenlinux.io) is supported.
* `$flavor`: Additional packages on top of the base image for the given purpose.
  E.g. the CAPI flavor contains packages necessary to deploy CAPI nodes.
* `$version`: The version of the base image, distribution specific.
* `$variant`: The platform this image runs on, currently `kvm` and `metal` are
  supported.
* `$arch`: Either `amd64` or `arch64`.
* `-manual`: Set for images built and published by hand.

The architecture and variant can also automatically be selected by fetching the
tag without them to get a manifest listing all permutations. Note that, if you
specify the variant or the architecture you also need to specify the other.
We automatically publish the following images:

```
ghcr.io/ironcore-dev/os-images
└── /gardenlinux
    ├── :$version
    ├── :$version-metal-amd64
    ├── :$version-metal-arm64
    ├── /gardener
    │   ├── :$version
    │   ├── :$version-kvm-amd64
    │   ├── :$version-kvm-arm64
    │   ├── :$version-metal-amd64
    │   └── :$version-metal-arm64
    └── /capi
        ├── :$version
        ├── :$version-metal-amd64
        └── :$version-metal-arm64
```

Since we are currently only re-publishing the gardenlinux images, we can not
publish the full matrix as gardenlinux builds only some of the combinations we'd
like to support.

## Workflow

The main publishing workflow (`publish-gardenlinux.yml`) runs weekly on Sundays
and can also be triggered manually via `workflow_dispatch`. It:

1. **Resolves the version** to publish: uses the manually provided version or
   discovers the latest Garden Linux release from the GitHub API. Scheduled runs
   skip publishing if the version has already been published.
2. **Downloads artifacts** from the Garden Linux GitHub release for both `amd64`
   and `arm64` architectures.
3. **Builds multi-arch OCI images** using
   [`ironcore-image`](https://github.com/ironcore-dev/ironcore-image) for each
   flavor and variant combination.
4. **Pushes images** to `ghcr.io` with per-variant and per-arch sub-manifest
   tags.

For metal images, the workflow also builds a Unified Kernel Image (UKI) using
`ukify` with the EFI stub extracted from the Garden Linux artifacts.

> [!NOTE]
>
> The per-variant tags (e.g. `:$version-metal-amd64`) are available today.
> The combined `:$version` index manifest listing all variant-arch permutations
> is not yet produced and will require extending `ironcore-image` with variant
> support.

## Documentation
- Manual build (Garden Linux -> `ironcore-image`): `docs/manual-oci-image-build.md`

## Contributing
Contributions to enhance and broaden the scope of the os-images project are encouraged. Please ensure all changes are well-tested before submission.

## License

This project is licensed under the [Apache License 2.0](LICENSE).
