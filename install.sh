#!/bin/bash
set -e

# Configuration
REPO="markshao/DevSwarm"
BINARY_NAME="devswarm"
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
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
else
    echo "  (Needs sudo permission to move binary)"
    sudo mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
fi

# 4. Setup Autocompletion (Preserved from previous script)
echo "🐚 Setting up autocompletion..."
SHELL_TYPE=$(basename "$SHELL")

if [ "$SHELL_TYPE" = "zsh" ]; then
    if ! grep -q "devswarm completion zsh" ~/.zshrc; then
        echo "  Adding completion to ~/.zshrc"
        echo '' >> ~/.zshrc
        echo '# DevSwarm Autocompletion' >> ~/.zshrc
        echo 'source <(devswarm completion zsh)' >> ~/.zshrc
        echo -e "${GREEN}✅ Added zsh completion. Please restart your terminal or run 'source ~/.zshrc'${NC}"
    else
        echo "  Zsh completion already configured."
    fi
elif [ "$SHELL_TYPE" = "bash" ]; then
    if ! grep -q "devswarm completion bash" ~/.bash_profile; then
        echo "  Adding completion to ~/.bash_profile"
        echo '' >> ~/.bash_profile
        echo '# DevSwarm Autocompletion' >> ~/.bash_profile
        echo 'source <(devswarm completion bash)' >> ~/.bash_profile
        echo -e "${GREEN}✅ Added bash completion. Please restart your terminal.${NC}"
    else
        echo "  Bash completion already configured."
    fi
else
    echo "  Shell '$SHELL_TYPE' not automatically supported for completion setup."
    echo "  You can manually add: source <(devswarm completion $SHELL_TYPE)"
fi

echo -e "${GREEN}✨ DevSwarm installed successfully! Run 'devswarm help' to get started.${NC}"
