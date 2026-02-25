# Hack Scripts

## build-metal-dev.sh

If you need to build a custom image (for example to set custom kernel parameters
via the kernel cmdline) use this. First download the baremetal PXE release asset
from gardenlinux, then run the script with it as the only argument. Example:

```
curl -sL "https://github.com/gardenlinux/gardenlinux/releases/download/2150.0.0/baremetal_pxe-amd64-2150.0.0-eb8696b9.tar.xz" -o "baremetal_pxe-amd64-2150.0.0-eb8696b9.tar.xz"
./build-metal-dev.sh ./baremetal_pxe-amd64-2150.0.0-eb8696b9.tar.xz
```

If you require a custom image that is not available as a pre-built artifact you
have to build the `_pxe` release artifact yourself.

Make sure to modify the script to your needs before doing this. It will push the
image with the `-manual` tag suffix and a randomly generated string to avoid
conflicts.

Requirements:
* Tools: ironcore-image, ukify, cpio, xz, xxd
* Logged in to ghcr.io

This script will inevitably drift from the workflow, have your LLM check for any
drift before using it.
