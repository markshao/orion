#!/bin/bash

set -e

REPO="bytedance/DevSwarm"
BINARY="devswarm"
DEST="/usr/local/bin"

# Detect OS and Arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

if [ "$ARCH" == "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" == "arm64" ] || [ "$ARCH" == "aarch64" ]; then
    ARCH="arm64"
else
    echo "Unsupported architecture: $ARCH"
    exit 1
fi

echo "Detected OS: $OS, Arch: $ARCH"

# Determine latest release URL
# We assume standard GoReleaser naming convention: devswarm_{os}_{arch}
# e.g., devswarm_darwin_arm64
ASSET_NAME="${BINARY}_${OS}_${ARCH}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${ASSET_NAME}"

echo "Downloading $BINARY from $DOWNLOAD_URL..."
if curl -fsSL -o "$BINARY" "$DOWNLOAD_URL"; then
    chmod +x "$BINARY"
    echo "Installing to $DEST (requires sudo)..."
    sudo mv "$BINARY" "$DEST/$BINARY"
    echo "Successfully installed $BINARY to $DEST/$BINARY"
    $BINARY --version
else
    echo "Failed to download binary. Please check if a release exists for your platform."
    echo "Alternatively, you can build from source (see README)."
    exit 1
fi
