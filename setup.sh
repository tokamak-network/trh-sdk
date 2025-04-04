#!/usr/bin/env bash
# Detect OS
if [[ "$OSTYPE" == "darwin"* ]]; then
    OS="MacOS"
elif [[ "$OSTYPE" == "linux"* ]]; then
    OS="Linux"
else
    echo "Unsupported operating system"
    exit 1
fi

echo "Detected OS: $OS"

# Save the current Go version
current_go_version=$(go version 2>/dev/null)

# Check if the current version is not v1.22.6
if ! echo "$current_go_version" | grep 'go1.22.6' &>/dev/null ; then
    # Download Go
    echo "Installing Go..."
    if [[ "$OS" == "Linux" ]]; then
        wget https://go.dev/dl/go1.22.6.linux-amd64.tar.gz
        # Remove existing Go installation and extract new one
        sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.22.6.linux-amd64.tar.gz
        # Clean up downloaded archive
        rm go1.22.6.linux-amd64.tar.gz
    elif [[ "$OS" == "MacOS" ]]; then
        wget https://go.dev/dl/go1.22.6.darwin-amd64.pkg
        # Install Go package
        sudo installer -pkg go1.22.6.darwin-amd64.pkg -target /
        # Clean up downloaded package
        rm go1.22.6.darwin-amd64.pkg
    fi
fi

# Set shell config file based on OS
if [[ "$OS" == "Linux" ]]; then
    SHELL_CONFIG="$HOME/.bashrc"
else
    SHELL_CONFIG="$HOME/.zshrc"
fi

# Add required PATH exports if not already present
if ! grep -q "export PATH=\$PATH:/usr/local/go/bin" "$SHELL_CONFIG"; then
    echo "export PATH=\$PATH:/usr/local/go/bin" >> "$SHELL_CONFIG"
fi

if ! grep -q "export PATH=\$HOME/go/bin:\$PATH" "$SHELL_CONFIG"; then
    echo "export PATH=\$HOME/go/bin:\$PATH" >> "$SHELL_CONFIG"
fi

if ! grep -q "export PATH=\$PATH:\$HOME/.foundry/bin" "$SHELL_CONFIG"; then
    echo "export PATH=\$PATH:\$HOME/.foundry/bin" >> "$SHELL_CONFIG"
fi

# Source shell config and set PATH temporarily for this session
source "$SHELL_CONFIG" || true
export PATH=$PATH:/usr/local/go/bin
export PATH=$HOME/go/bin:$PATH

echo "✅ Go has been installed successfully!"
# Verify Go installation
echo "Verifying Go installation..."
/usr/local/go/bin/go version


# Install TRH SDK CLI
echo "Installing TRH SDK CLI..."
# Use full path to go binary since PATH may not be updated yet
/usr/local/go/bin/go install github.com/tokamak-network/trh-sdk@9fdf8aaf2a0607b43f3b87aac4e24bc598b97bb0

echo "✅ TRH SDK has been installed successfully!"

