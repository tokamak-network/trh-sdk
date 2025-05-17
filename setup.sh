#!/usr/bin/env bash

# Detect current shell
CURRENT_SHELL=$(ps -p $$ -o comm=)

# Re-run with the correct interpreter depending on the OS
# Use SKIP_SHEBANG_CHECK variable to prevent infinite loop if already re-run
if [ "$(uname)" = "Darwin" ] && [ -z "$SKIP_SHEBANG_CHECK" ]; then
    if [ " $CURRENT_SHELL" != "zsh" ]; then
        export SKIP_SHEBANG_CHECK=1
        echo "macOS detected. Current shell: $CURRENT_SHELL.Switching to zsh interpreter......"
        exec /bin/zsh "$0" "$@"
    fi
fi

# Detect OS
# Detect Operating System
OS_TYPE=$(uname)

# Detect Architecture
ARCH=$(uname -m)

if [[ "$ARCH" == "x86_64" ]]; then
    ARCH="amd64"
elif [[ "$ARCH" == "aarch64" || "$ARCH" == "arm64" ]]; then
    ARCH="arm64"
elif [[ "$ARCH" == "armv6l" ]]; then
    ARCH="armv6l"
elif [[ "$ARCH" == "i386" ]]; then
    ARCH="386"
else
    echo "$ARCH is an unsupported architecture."
    exit 1
fi

echo

# MacOS
if [[ "$OS_TYPE" == "Darwin" ]]; then
    OS_TYPE="darwin"
    echo "🚀 Starting package installation for MacOS $ARCH!"

# Linux
elif [[ "$OS_TYPE" == "Linux" ]]; then
    OS_TYPE="linux"
    # Detect the Linux distribution (Ubuntu, Fedora, Arch, etc)
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        OS_NAME=$NAME
        echo "🚀 Starting package installation for Linux/$OS_NAME $ARCH!"
    fi

# Other OS (Windows, BSD, etc.)
else
    echo "$OS_TYPE is an unsupported operating system."
    exit 1
fi

echo


# Check Shell
SHELL_NAME=$(basename "$SHELL")
if [[ "$SHELL_NAME" == "zsh" ]]; then
    echo "The current shell is $SHELL_NAME. The installation will proceed based on $SHELL_NAME."
elif [[ "$SHELL_NAME" == "bash" ]]; then
    echo "The current shell is $SHELL_NAME. The installation will proceed based on $SHELL_NAME."
else
    echo "The current shell is $SHELL_NAME. $SHELL_NAME is an unsupported shell."
    exit 1
fi

# Set Config File
if [ "$SHELL_NAME" = "zsh" ]; then
    CONFIG_FILE="$HOME/.zshrc"
    PROFILE_FILE="$HOME/.zshrc"
    # check config & profile
    echo "The shell name is $SHELL_NAME"
    echo "The config file name is $CONFIG_FILE"
    echo "The profile name is $PROFILE_FILE"
elif [ "$SHELL_NAME" = "bash" ]; then
    CONFIG_FILE="$HOME/.bashrc"
    PROFILE_FILE="$HOME/.profile"
fi



# Setup Go version
# MacOS specific steps
if [[ "$OS_TYPE" == "darwin" ]]; then
    # 1. Install Homebrew
    echo "Installing Homebrew..."
    if ! command -v brew &> /dev/null; then
        echo "Homebrew not found, installing..."
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

        if [[ "$ARCH" == "aarch64" || "$ARCH" == "arm64" ]]; then
            # Check if the Homebrew configuration is already in the CONFIG_FILE
            if ! grep -Fxq 'export PATH="/usr/local/bin:$PATH"' "$CONFIG_FILE"; then
                # If the configuration is not found, add Homebrew to the current shell session
                {
                    echo ''
                    echo 'export PATH="/usr/local/bin:$PATH"'
                } >> "$CONFIG_FILE"
            fi

            # Check if the Homebrew configuration is already in the PROFILE_FILE
            if ! grep -Fxq 'export PATH="/usr/local/bin:$PATH"' "$PROFILE_FILE"; then
                # If the configuration is not found, add Homebrew to the current shell session
                {
                    echo ''
                    echo 'export PATH="/usr/local/bin:$PATH"'
                } >> "$PROFILE_FILE"
            fi
        fi

        export PATH="/opt/homebrew/bin:$PATH"
    else
        echo "Homebrew is already installed."
    fi

    STEP=$((STEP + 1))
    echo

    # 2. Install Git
    echo "Installing Git..."
    if ! command -v git &> /dev/null; then
        echo "git not found, installing..."
        brew install git
    else
        echo "git is already installed."
    fi

    STEP=$((STEP + 1))
    echo

    # 3. Install Xcode Command Line Tools(Inclue make)
    echo "Installing Xcode Command Line Tools..."
    if ! xcode-select -p &> /dev/null; then
        echo "Xcode Command Line Tools not found, installing..."
        xcode-select --install
    else
        echo "Xcode Command Line Tools are already installed."
    fi

    STEP=$((STEP + 1))
    echo

    # 4. Install Go (v1.22.6)
    # 4-1. Install Go (v1.22.6)
    echo "Installing Go (v1.22.6)..."
    export PATH="$PATH:/usr/local/go/bin"

    # Save the current Go version
    current_go_version=$(go version 2>/dev/null)

    # Check if the current version is not v1.22.6
    if ! echo "$current_go_version" | grep 'go1.22.6' &>/dev/null ; then
        # If Go is installed, remove it
        if command -v go &> /dev/null; then
            echo "Go is already installed. Removing the existing version..."
            sudo rm -rf "$(which go)"
        fi

        if ! command -v curl &> /dev/null; then
            echo "curl not found, installing..."
            brew install curl
        else
            echo "curl is already installed."
        fi

        GO_FILE_NAME="go1.22.6.darwin-${ARCH}.tar.gz"
        GO_DOWNLOAD_URL="https://go.dev/dl/${GO_FILE_NAME}"

        sudo curl -L -o "${GO_FILE_NAME}" "${GO_DOWNLOAD_URL}"

        sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf "${GO_FILE_NAME}"

        # Check if the Go configuration is already in the CONFIG_FILE
        if ! grep -Fxq 'export PATH="$PATH:/usr/local/go/bin"' "$CONFIG_FILE"; then
            # If the configuration is not found, add Go to the current shell session
            {
                echo ''
                echo 'export PATH="$PATH:/usr/local/go/bin"'
            } >> "$CONFIG_FILE"
        fi

        # Check if the NVM configuration is already in the PROFILE_FILE
        if ! grep -Fxq 'export PATH=$PATH:/usr/local/go/bin' "$PROFILE_FILE"; then
            # If the configuration is not found, add Go to the current shell session
            {
                echo ''
                echo 'export PATH="$PATH:/usr/local/go/bin"'
            } >> "$PROFILE_FILE"
        fi

        export PATH="$PATH:/usr/local/go/bin"
    else
        echo "Go 1.22.6 is already installed."
    fi

echo $OS_TYPE

# Linux specific steps
elif [[ "$OS_TYPE" == "linux" ]]; then
    # If the operating system is Ubuntu, execute the following commands
    if [[ "$OS_NAME" == "Ubuntu" ]]; then

        if ! command -v sudo &> /dev/null; then
            echo "sudo not found, installing..."
            apt-get install -y sudo
        else
            echo "sudo is already installed."
        fi

        # 1. Update package list
        echo "Updating package list..."
        sudo apt-get update -y

        STEP=$((STEP + 1))
        echo

        # 2. Install Git
        echo "Installing Git..."
        if ! command -v git &> /dev/null; then
            echo "git not found, installing..."
            sudo apt-get install -y git
        else
            echo "git is already installed."
        fi

        STEP=$((STEP + 1))
        echo

        # 3. Install Build-essential
        echo "Installing Build-essential..."
        if ! dpkg -s build-essential &> /dev/null; then
            echo "Build-essential not found, installing..."
            sudo apt-get install -y build-essential
        else
            echo "Build-essential is already installed."
        fi

        STEP=$((STEP + 1))
        echo

        # 4. Install Go (v1.22.6)
        # 4-1. Install Go (v1.22.6)
        echo "Installing Go (v1.22.6)..."
        export PATH="$PATH:/usr/local/go/bin"

        # Save the current Go version
        current_go_version=$(go version 2>/dev/null)

        # Check if the current version is not v1.22.6
        if ! echo "$current_go_version" | grep 'go1.22.6' &>/dev/null ; then

            echo "Installing go1.22.6..."
            # If Go is installed, remove it
            if command -v go &> /dev/null; then
                echo "Go is already installed. Removing the existing version..."
                sudo rm -rf "$(which go)"
            fi

            if ! command -v curl &> /dev/null; then
                echo "curl not found, installing..."
                sudo apt-get install -y curl
            else
                echo "curl is already installed."
            fi

            GO_FILE_NAME="go1.22.6.linux-${ARCH}.tar.gz"
            GO_DOWNLOAD_URL="https://go.dev/dl/${GO_FILE_NAME}"

            sudo curl -L -o "${GO_FILE_NAME}" "${GO_DOWNLOAD_URL}"

            sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf "${GO_FILE_NAME}"

            # Check if the Go configuration is already in the CONFIG_FILE
            if ! grep -Fxq 'export PATH="$PATH:/usr/local/go/bin"' "$CONFIG_FILE"; then
                # If the configuration is not found, add Go to the current shell session
                {
                    echo ''
                    echo 'export PATH="$PATH:/usr/local/go/bin"'
                } >> "$CONFIG_FILE"
            fi

            # Check if the NVM configuration is already in the PROFILE_FILE
            if ! grep -Fxq 'export PATH=$PATH:/usr/local/go/bin' "$PROFILE_FILE"; then
                # If the configuration is not found, add Go to the current shell session
                {
                    echo ''
                    echo 'export PATH="$PATH:/usr/local/go/bin"'
                } >> "$PROFILE_FILE"
            fi

            export PATH="$PATH:/usr/local/go/bin"
        else
            echo "Go 1.22.6 is already installed."
        fi

        STEP=$((STEP + 1))
        echo

        SUCCESS="true"
        echo

    # If it is an operating system other than Ubuntu, execute the following commands.
    else
        echo "$OS_NAME is an unsupported operating system."
    fi
fi



# Add required PATH exports if not already present
if ! grep -q "export PATH=\$PATH:/usr/local/go/bin" "$CONFIG_FILE"; then
    echo "export PATH=\$PATH:/usr/local/go/bin" >> "$CONFIG_FILE"
fi

if ! grep -q "export PATH=\$HOME/go/bin:\$PATH" "$CONFIG_FILE"; then
    echo "export PATH=\$HOME/go/bin:\$PATH" >> "$CONFIG_FILE"
fi

if ! grep -q "export PATH=\$PATH:\$HOME/.foundry/bin" "$CONFIG_FILE"; then
    echo "export PATH=\$PATH:\$HOME/.foundry/bin" >> "$CONFIG_FILE"
fi

if ! grep -q "export PATH=\"\$HOME/.local/share/pnpm:\$PATH\"" "$CONFIG_FILE"; then
    echo "export PATH=\"\$HOME/.local/share/pnpm:\$PATH\"" >> "$CONFIG_FILE"
fi
if ! grep -q "export PATH=\"\$HOME/.cargo/env:\$PATH\"" "$CONFIG_FILE"; then
    echo "export PATH=\"\$HOME/.cargo/env:\$PATH\"" >> "$CONFIG_FILE"
fi

# Source shell config and set PATH temporarily for this session
if [ "$SHELL_NAME" = "zsh" ]; then
    zsh -c source "$CONFIG_FILE"
else
    bash -c source "$CONFIG_FILE"
fi
echo $CONFIG_FILE

echo "✅ Go has been installed successfully!"
# Verify Go installation
echo "Verifying Go installation..."
go version


# Install TRH SDK CLI
echo "Installing TRH SDK CLI..."
# Use full path to go binary since PATH may not be updated yet
go install github.com/tokamak-network/trh-sdk@latest

echo "✅ TRH SDK has been installed successfully!"

echo "Verifying TRH SDK installation..."
"$(go env GOPATH)/bin/trh-sdk" version

echo -e "\033[1;32msource $CONFIG_FILE\033[0m to apply changes to your current shell session."