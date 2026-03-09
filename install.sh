#!/bin/bash

set -e

REPO="markshao/orion"
BINARY="orion"
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
# We assume standard GoReleaser naming convention: orion_{os}_{arch}.tar.gz
# e.g., orion_darwin_arm64.tar.gz
ASSET_NAME="${BINARY}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${ASSET_NAME}"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading $BINARY from $DOWNLOAD_URL..."
if curl -fsSL -o "$TMP_DIR/$ASSET_NAME" "$DOWNLOAD_URL"; then
    echo "Extracting..."
    tar -xzf "$TMP_DIR/$ASSET_NAME" -C "$TMP_DIR"
    
    if [ -f "$TMP_DIR/$BINARY" ]; then
        echo "Installing to $DEST (requires sudo)..."
        sudo mv "$TMP_DIR/$BINARY" "$DEST/$BINARY"
        chmod +x "$DEST/$BINARY"
        echo "Successfully installed $BINARY to $DEST/$BINARY"
        $BINARY --version

        # Autocompletion setup
        echo "Setting up autocompletion..."
        SHELL_TYPE=$(basename "$SHELL")
        
        if [ "$SHELL_TYPE" == "zsh" ]; then
             # Zsh completion
            COMPLETION_FILE="${HOME}/.orion_completion.zsh"
            $BINARY completion zsh > "$COMPLETION_FILE"
            
            if ! grep -q "source $COMPLETION_FILE" "${HOME}/.zshrc"; then
                echo "" >> "${HOME}/.zshrc"
                echo "# Orion autocompletion" >> "${HOME}/.zshrc"
                echo "source $COMPLETION_FILE" >> "${HOME}/.zshrc"
                echo "Added Zsh completion to ~/.zshrc. Please restart your terminal."
            else
                echo "Zsh completion already configured."
            fi
            
        elif [ "$SHELL_TYPE" == "bash" ]; then
             # Bash completion
            COMPLETION_FILE="${HOME}/.orion_completion.bash"
            $BINARY completion bash > "$COMPLETION_FILE"
            
            RC_FILE="${HOME}/.bashrc"
            if [ -f "${HOME}/.bash_profile" ]; then
                RC_FILE="${HOME}/.bash_profile"
            fi
            
            if ! grep -q "source $COMPLETION_FILE" "$RC_FILE"; then
                echo "" >> "$RC_FILE"
                echo "# Orion autocompletion" >> "$RC_FILE"
                echo "source $COMPLETION_FILE" >> "$RC_FILE"
                echo "Added Bash completion to $RC_FILE. Please restart your terminal."
            else
                echo "Bash completion already configured."
            fi
        else
            echo "Skipping autocompletion: unsupported shell '$SHELL_TYPE'. Manual setup required."
            echo "Run '$BINARY completion <shell>' to generate script."
        fi
    else
        echo "Error: Binary '$BINARY' not found in archive."
        ls -l "$TMP_DIR"
        exit 1
    fi
else
    echo "Failed to download binary. Please check if a release exists for your platform."
    echo "URL: $DOWNLOAD_URL"
    echo "Alternatively, you can build from source (see README)."
    exit 1
fi
