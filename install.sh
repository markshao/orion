#!/bin/bash
set -e

# Configuration
BINARY_NAME="devswarm"
INSTALL_DIR="/usr/local/bin"
GO_MOD_NAME="devswarm" # Should match go.mod

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}🐝 Installing DevSwarm...${NC}"

# 1. Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed. Please install Go first.${NC}"
    exit 1
fi

# 2. Build the binary
echo "📦 Building binary..."
if [ -f "go.mod" ]; then
    go build -o "$BINARY_NAME" main.go
else
    echo -e "${RED}Error: go.mod not found. Please run this script from the project root.${NC}"
    exit 1
fi

# 3. Install binary to /usr/local/bin (requires sudo)
echo "🚀 Installing to $INSTALL_DIR..."
if [ -w "$INSTALL_DIR" ]; then
    mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
else
    echo "  (Needs sudo permission to move binary)"
    sudo mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
fi

# 4. Setup Autocompletion
echo "🐚 Setting up autocompletion..."
SHELL_TYPE=$(basename "$SHELL")

if [ "$SHELL_TYPE" = "zsh" ]; then
    # Zsh completion
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
    # Bash completion (macOS default bash might be old, but try anyway)
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
