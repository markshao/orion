#!/bin/bash
set -e

# Configuration
REPO="markshao/DevSwarm"
BINARY_NAME="ds"
INSTALL_DIR="/usr/local/bin"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}🐝 Installing DevSwarm...${NC}"

# 1. Detect OS and Arch
OS=$(uname -s)
ARCH=$(uname -m)

# Standardize OS name (Darwin/Linux)
if [ "$OS" = "Darwin" ]; then
    OS="Darwin"
elif [ "$OS" = "Linux" ]; then
    OS="Linux"
else
    echo -e "${RED}Error: Unsupported OS: $OS${NC}"
    exit 1
fi

# Standardize Arch name (x86_64/arm64)
# Maps to Goreleaser's template:
# amd64 -> x86_64
# arm64 -> arm64
if [ "$ARCH" = "x86_64" ]; then
    ARCH="x86_64"
elif [ "$ARCH" = "aarch64" ]; then
    ARCH="arm64"
elif [ "$ARCH" = "arm64" ]; then
    ARCH="arm64"
else
    echo -e "${RED}Error: Unsupported Architecture: $ARCH${NC}"
    exit 1
fi

# 2. Construct Download URL
# Pattern: devswarm_Darwin_arm64.tar.gz
ASSET_NAME="${BINARY_NAME}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${ASSET_NAME}"

echo "⬇️  Downloading ${ASSET_NAME} from GitHub..."

# Create a temporary directory
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

# Download using curl
if ! curl -sL -o "$TMP_DIR/$ASSET_NAME" "$DOWNLOAD_URL"; then
    echo -e "${RED}Error: Failed to download release asset.${NC}"
    echo "Check if the release exists: $DOWNLOAD_URL"
    exit 1
fi

# Extract
echo "📦 Extracting..."
tar -xzf "$TMP_DIR/$ASSET_NAME" -C "$TMP_DIR"

# 3. Install binary
echo "🚀 Installing to $INSTALL_DIR..."
mv "$TMP_DIR/ds" "$INSTALL_DIR/ds"
chmod +x "$INSTALL_DIR/ds"

# 4. Setup Autocomplete
SHELL_TYPE=$(basename "$SHELL")

if [ "$SHELL_TYPE" = "zsh" ]; then
    echo "⚙️  Configuring zsh completion..."
    if ! grep -q "ds completion zsh" ~/.zshrc; then
        echo 'source <(ds completion zsh)' >> ~/.zshrc
        echo "  Added completion to ~/.zshrc"
    else
        echo "  Completion already exists in ~/.zshrc"
    fi
elif [ "$SHELL_TYPE" = "bash" ]; then
    echo "⚙️  Configuring bash completion..."
    if ! grep -q "ds completion bash" ~/.bashrc; then
        echo 'source <(ds completion bash)' >> ~/.bashrc
        echo "  Added completion to ~/.bashrc"
    else
        echo "  Completion already exists in ~/.bashrc"
    fi
else
    echo "  Shell '$SHELL_TYPE' not automatically supported for completion setup."
    echo "  You can manually add: source <(ds completion $SHELL_TYPE)"
fi

echo -e "${GREEN}✨ DevSwarm installed successfully! Run 'ds help' to get started.${NC}"
