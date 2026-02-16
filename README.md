# os-images

[![REUSE status](https://api.reuse.software/badge/github.com/ironcore-dev/os-images)](https://api.reuse.software/info/github.com/ironcore-dev/os-images)
[![GitHub License](https://img.shields.io/static/v1?label=License&message=Apache-2.0&color=blue)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://makeapullrequest.com)

## Overview
The `os-images` repository specializes in automating the release and publication of Operating System (OS) image artifacts as OCI (Open Container Initiative) images. This project streamlines the process of pushing these OS images to the project's GitHub Container Registry (`ghcr.io`) via GitHub Actions.

## Workflow
The GitHub Actions workflow triggers on pushes to the `main` branch and performs the following tasks:
- **Download and Extract OS Artifact**: Automates the process of downloading and extracting specified OS artifacts.
- **Prepare and Push Images with ORAS**: Pushes the prepared OS images to `ghcr.io`, tagging each with its specific version and also as `latest`.

## Configuration
The configuration for OS artifacts is managed through the `os_image_artifacts.yml` file located in the `.github` directory. Currently only `amd64` and `arm64` platforms are supported.

## Documentation
- Manual build (Garden Linux -> `ironcore-image`): `docs/manual-oci-image-build.md`

### Example Configuration
```yaml
amd64:
  gardenlinux_kvm_artifact_url: https://github.com/gardenlinux/gardenlinux/releases/download/1592.3/kvm-gardener_prod-amd64-1592.3-f64e280f.tar.xz
  gardenlinux_metal_artifact_url: https://github.com/gardenlinux/gardenlinux/releases/download/1592.3/metal-gardener_prod_pxe-amd64-1592.3-f64e280f.tar.xz
arm64:
  gardenlinux_kvm_artifact_url: https://github.com/gardenlinux/gardenlinux/releases/download/1592.3/kvm-gardener_prod-arm64-1592.3-f64e280f.tar.xz
  gardenlinux_metal_artifact_url: https://github.com/gardenlinux/gardenlinux/releases/download/1592.3/metal-gardener_prod_pxe-arm64-1592.3-f64e280f.tar.xz
```

## Contributing
Contributions to enhance and broaden the scope of the os-images project are encouraged. Please ensure all changes are well-tested before submission.

## License

This project is licensed under the [Apache License 2.0](LICENSE).
