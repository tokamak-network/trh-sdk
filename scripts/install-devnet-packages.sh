#!/usr/bin/env bash

TOTAL_STEPS=10
STEP=1
SUCCESS="false"

# Detect Operating System
OS_TYPE=$(uname)

# Detect current shell
CURRENT_SHELL=$(ps -p $$ -o comm=)

# Re-run with the correct interpreter depending on the OS
# Use SKIP_SHEBANG_CHECK variable to prevent infinite loop if already re-run
if [ "$OS_TYPE" = "Darwin" ] && [ -z "$SKIP_SHEBANG_CHECK" ]; then
    if [ "$CURRENT_SHELL" != "zsh" ]; then
        if [ -x "/bin/zsh" ]; then
            export SKIP_SHEBANG_CHECK=1
            echo "macOS detected. Current shell: $CURRENT_SHELL. Switching to zsh interpreter......"
            exec /bin/zsh "$0" "$@"
        else
            echo "Error: /bin/zsh not found. Please ensure zsh is installed." >&2
            exit 1
        fi
    fi
fi

# Detect Architecture
ARCH=$(uname -m)

if [[ "$ARCH" == "x86_64" ]] || [[ "$ARCH" == "amd64" ]]; then
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
elif [ "$SHELL_NAME" = "bash" ]; then
    CONFIG_FILE="$HOME/.bashrc"
    PROFILE_FILE="$HOME/.profile"
fi

echo

# Function to display completion message
function display_completion_message {
    if [[ "$SUCCESS" == "true" ]]; then
        echo ""
        echo "All $TOTAL_STEPS steps are complete."
        echo ""
        exit 0
    else
        echo ""
        echo "Installation was interrupted. Completed $((STEP - 1))/$TOTAL_STEPS steps."

        echo ""
        echo "Please source your profile to apply changes:"
        echo -e "\033[1;32msource $CONFIG_FILE\033[0m"
        exit 1
    fi
}

# Use trap to display message on script exit, whether successful or due to an error
trap display_completion_message EXIT
trap "echo 'Process interrupted!'; exit 1" INT

# MacOS specific steps
if [[ "$OS_TYPE" == "darwin" ]]; then

    # 1. Install Homebrew
    echo "[$STEP/$TOTAL_STEPS] ----- Installing Homebrew..."
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
    echo "[$STEP/$TOTAL_STEPS] ----- Installing Git..."
    if ! command -v git &> /dev/null; then
        echo "git not found, installing..."
        brew install git
    else
        echo "git is already installed."
    fi

    STEP=$((STEP + 1))
    echo

    # 3. Install or Upgrade Xcode Command Line Tools (Include make)
    echo "[$STEP/$TOTAL_STEPS] ----- Installing or Upgrading Xcode Command Line Tools..."
    if ! xcode-select -p &> /dev/null; then
        echo "Xcode Command Line Tools not found, installing..."
        xcode-select --install
    else
        echo "Xcode Command Line Tools are already installed. Checking for updates..."
    fi

    STEP=$((STEP + 1))
    echo

    # 4. Install Node.js (v20.16.0)
    echo "[$STEP/$TOTAL_STEPS] ----- Installing Node.js (v20.16.0)..."

    # Save the current Node.js version
    current_node_version=$(node -v 2>/dev/null)

    # Check if the current version is not v20.16.0
    if [[ "$current_node_version" != "v20.16.0" ]]; then

        # 4-1. Install NVM
        echo "[$STEP/$TOTAL_STEPS] ----- Installing NVM..."

        # Create NVM directory if it doesn't exist
        export NVM_DIR="$HOME/.nvm"
        mkdir -p "$NVM_DIR"
        HOMEBREW_PREFIX=$(brew --prefix)
        [ -s "$HOMEBREW_PREFIX/opt/nvm/nvm.sh" ] && \. "$HOMEBREW_PREFIX/opt/nvm/nvm.sh"
        [ -s "$HOMEBREW_PREFIX/opt/nvm/etc/bash_completion.d/nvm" ] && \. "$HOMEBREW_PREFIX/opt/nvm/etc/bash_completion.d/nvm"

        if ! command -v nvm &> /dev/null; then
            echo "NVM not found, installing..."
            brew install nvm

            # Check if the NVM configuration is already in the CONFIG_FILE
            if ! grep -Fxq 'export NVM_DIR="$HOME/.nvm"' "$CONFIG_FILE"; then

                # If the configuration is not found, add NVM to the current shell session
                {
                    echo ''
                    echo 'export NVM_DIR="$HOME/.nvm"'
                    echo "[ -s \"$HOMEBREW_PREFIX/opt/nvm/nvm.sh\" ] && \. \"$HOMEBREW_PREFIX/opt/nvm/nvm.sh\""
                    echo "[ -s \"$HOMEBREW_PREFIX/opt/nvm/etc/bash_completion.d/nvm\" ] && \. \"$HOMEBREW_PREFIX/opt/nvm/etc/bash_completion.d/nvm\""
                } >> "$CONFIG_FILE"
            fi

            # Check if the NVM configuration is already in the PROFILE_FILE
            if ! grep -Fxq 'export NVM_DIR="$HOME/.nvm"' "$PROFILE_FILE"; then

                # If the configuration is not found, add NVM to the current shell session
                {
                    echo ''
                    echo 'export NVM_DIR="$HOME/.nvm"'
                    echo "[ -s \"$HOMEBREW_PREFIX/opt/nvm/nvm.sh\" ] && \. \"$HOMEBREW_PREFIX/opt/nvm/nvm.sh\""
                    echo "[ -s \"$HOMEBREW_PREFIX/opt/nvm/etc/bash_completion.d/nvm\" ] && \. \"$HOMEBREW_PREFIX/opt/nvm/etc/bash_completion.d/nvm\""
                } >> "$PROFILE_FILE"
            fi

            [ -s "$HOMEBREW_PREFIX/opt/nvm/nvm.sh" ] && \. "$HOMEBREW_PREFIX/opt/nvm/nvm.sh"
            [ -s "$HOMEBREW_PREFIX/opt/nvm/etc/bash_completion.d/nvm" ] && \. "$HOMEBREW_PREFIX/opt/nvm/etc/bash_completion.d/nvm"
        else
            echo "NVM is already installed."
        fi

        # 4-2. Install Node.js v20.16.0 using NVM
        echo "[$STEP/$TOTAL_STEPS] ----- Installing Node.js v20.16.0 using NVM..."
        if ! nvm ls | grep 'v20.16.0' | grep -v 'default' &> /dev/null; then
            echo "Node.js v20.16.0 not found, installing..."
            nvm install v20.16.0
        else
            echo "Node.js v20.16.0 is already installed."
        fi

        # 4-3. Set Node.js v20.16.0 as the default version
        echo "[$STEP/$TOTAL_STEPS] ----- Setting Node.js v20.16.0 as the default version..."
        echo "Switching to Node.js v20.16.0..."
        nvm use v20.16.0
        nvm alias default v20.16.0
        echo "Node.js v20.16.0 is now set as the default version."
    else
        echo "Node.js is already v20.16.0."
    fi

    STEP=$((STEP + 1))
    echo

    # 5. Install Pnpm
    echo "[$STEP/$TOTAL_STEPS] ----- Installing Pnpm..."
    if ! command -v pnpm &> /dev/null; then
        echo "pnpm not found, installing..."
        brew install pnpm
    else
        echo "pnpm is already installed."
    fi

    STEP=$((STEP + 1))
    echo

    # 6. Install and Run Docker
    echo "[$STEP/$TOTAL_STEPS] ----- Installing Docker Engine..."
    if ! command -v docker &> /dev/null; then
        echo "Docker not found, installing..."
        brew install --cask docker
    else
        echo "Docker is already installed."
    fi

    # Run Docker Daemon
    echo "Starting Docker Daemon..."
    open -a Docker
    sudo chmod 666 /var/run/docker.sock

    echo "[$STEP/$TOTAL_STEPS] ----- Installing Docker Compose..."
    if ! command -v docker-compose &> /dev/null; then
        echo "Docker Compose not found, installing..."
        brew install docker-compose
    else
        echo "Docker Compose is already installed."
    fi

    STEP=$((STEP + 1))
    echo

    # 7. Install Foundry
    echo "[$STEP/$TOTAL_STEPS] ----- Installing Foundry..."
    echo "Installing Foundry..."

    # Check if jq is installed
    if ! command -v jq &> /dev/null; then
        echo "jq not found, installing..."
        sudo brew install -y jq
    else
        echo "✅ jq is already installed"
    fi

    # Check if Foundry is already installed with expected version
    if forge --version &> /dev/null && cast --version &> /dev/null; then
        echo "✅ Foundry is already installed"
    else
        # Install Foundry
        echo "Installing/updating Foundry..."
        if ! command -v curl &> /dev/null; then
            echo "curl not found, installing..."
            sudo brew install -y curl
            source $HOME/.zshrc
        fi
        # Install foundryup if not already installed
        if ! command -v foundryup &> /dev/null; then
            echo "Installing foundryup..."
            curl -L https://foundry.paradigm.xyz | bash
            source $HOME/.zshrc
        fi
        # Install stable version of Foundry
        if foundryup --install stable; then
            echo "✅ Foundry has been installed successfully!"
            forge --version
            cast --version 
            anvil --version
        else
            echo "❌ Foundry installation failed"
            exit 1
        fi
    fi

    SUCCESS="true"
    echo

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
        echo "[$STEP/$TOTAL_STEPS] ----- Updating package list..."
        sudo apt-get update -y

        STEP=$((STEP + 1))
        echo

        # 2. Install Git
        echo "[$STEP/$TOTAL_STEPS] ----- Installing Git..."
        if ! command -v git &> /dev/null; then
            echo "git not found, installing..."
            sudo apt-get install -y git
        else
            echo "git is already installed."
        fi

        STEP=$((STEP + 1))
        echo

        # 3. Install Build-essential
        echo "[$STEP/$TOTAL_STEPS] ----- Installing Build-essential..."
        if ! dpkg -s build-essential &> /dev/null; then
            echo "Build-essential not found, installing..."
            sudo apt-get install -y build-essential
        else
            echo "Build-essential is already installed."
        fi

        STEP=$((STEP + 1))
        echo

        # 5. Install Node.js (v20.16.0)
        echo "[$STEP/$TOTAL_STEPS] ----- Installing Node.js (v20.16.0)..."

        # Save the current Node.js version
        current_node_version=$(node -v 2>/dev/null)

        # Check if the current version is not v20.16.0
        if [[ "$current_node_version" != "v20.16.0" ]]; then

            # 5-1. Install NVM
            echo "[$STEP/$TOTAL_STEPS] ----- Installing NVM..."

            # Create NVM directory if it doesn't exist
            export NVM_DIR="$HOME/.nvm"
            mkdir -p "$NVM_DIR"
            [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
            [ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"

            if ! command -v nvm &> /dev/null; then
                echo "NVM not found, installing..."
                sudo curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash

                # Check if the NVM configuration is already in the CONFIG_FILE
                if ! grep -Fxq 'export NVM_DIR="$HOME/.nvm"' "$CONFIG_FILE"; then

                    # If the configuration is not found, add NVM to the current shell session
                    {
                        echo ''
                        echo 'export NVM_DIR="$HOME/.nvm"'
                        echo "[ -s \"$NVM_DIR/nvm.sh\" ] && \. \"$NVM_DIR/nvm.sh\""
                        echo "[ -s \"$NVM_DIR/bash_completion\" ] && \. \"$NVM_DIR/bash_completion\""
                    } >> "$CONFIG_FILE"
                fi

                # Check if the NVM configuration is already in the PROFILE_FILE
                if ! grep -Fxq 'export NVM_DIR="$HOME/.nvm"' "$PROFILE_FILE"; then

                    # If the configuration is not found, add NVM to the current shell session
                    {
                        echo ''
                        echo 'export NVM_DIR="$HOME/.nvm"'
                        echo "[ -s \"$NVM_DIR/nvm.sh\" ] && \. \"$NVM_DIR/nvm.sh\""
                        echo "[ -s \"$NVM_DIR/bash_completion\" ] && \. \"$NVM_DIR/bash_completion\""
                    } >> "$PROFILE_FILE"
                fi

                [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
                [ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"
            else
                echo "NVM is already installed."
            fi

            # 5-2. Install Node.js v20.16.0 using NVM
            echo "[$STEP/$TOTAL_STEPS] ----- Installing Node.js v20.16.0 using NVM..."
            if ! nvm ls | grep 'v20.16.0' | grep -v 'default' &> /dev/null; then
                echo "Node.js v20.16.0 not found, installing..."
                nvm install v20.16.0
            else
                echo "Node.js v20.16.0 is already installed."
            fi

            # 5-3. Set Node.js v20.16.0 as the default version
            echo "[$STEP/$TOTAL_STEPS] ----- Setting Node.js v20.16.0 as the default version..."
            echo "Switching to Node.js v20.16.0..."
            nvm use v20.16.0
            nvm alias default v20.16.0
            echo "Node.js v20.16.0 is now set as the default version."
        else
            echo "Node.js is already v20.16.0."
        fi

        STEP=$((STEP + 1))
        echo

        # 6. Install Pnpm
        echo "[$STEP/$TOTAL_STEPS] ----- Installing Pnpm..."
        export PATH="$HOME/.local/share/pnpm:$PATH"
        if ! command -v pnpm &> /dev/null; then
            echo "pnpm not found, installing..."
            curl -fsSL https://get.pnpm.io/install.sh | bash -

            # Check if the pnpm configuration is already in the CONFIG_FILE
            if ! grep -Fq 'export PATH="$HOME/.local/share/pnpm:$PATH"' "$CONFIG_FILE"; then

                # If the configuration is not found, add pnpm to the current shell session
                {
                    echo ''
                    echo 'export PATH="$HOME/.local/share/pnpm:$PATH"'
                } >> "$CONFIG_FILE"
            fi

            # Check if the pnpm configuration is already in the PROFILE_FILE
            if ! grep -Fq 'export PATH="$HOME/.local/share/pnpm:$PATH"' "$PROFILE_FILE"; then

                # If the configuration is not found, add pnpm to the current shell session
                {
                    echo ''
                    echo 'export PATH="$HOME/.local/share/pnpm:$PATH"'
                } >> "$PROFILE_FILE"
            fi

            export PATH="$HOME/.local/share/pnpm:$PATH"
        else
            echo "pnpm is already installed."
        fi

        STEP=$((STEP + 1))
        echo

        # 7. Install Foundry
        echo "[$STEP/$TOTAL_STEPS] ----- Installing Foundry..."
        echo "Installing Foundry..."

        # Check if jq is installed
        if ! command -v jq &> /dev/null; then
            echo "jq not found, installing..."
            sudo apt-get install -y jq
        else
            echo "✅ jq is already installed"
        fi

        # Check if Foundry is already installed with expected version
        if forge --version &> /dev/null && cast --version &> /dev/null; then
            echo "✅ Foundry is already installed"
        else
            # Install Foundry
            echo "Installing/updating Foundry..."
            if ! command -v curl &> /dev/null; then
                echo "curl not found, installing..."
                sudo apt-get install -y curl
            fi
            # Install foundryup if not already installed
            if ! command -v foundryup &> /dev/null; then
                echo "Installing foundryup..."
                curl -L https://foundry.paradigm.xyz | bash
                source $HOME/.bashrc
            fi
            # Install stable version of Foundry
            if foundryup --install stable; then
                echo "✅ Foundry has been installed successfully!"
                forge --version
                cast --version 
                anvil --version
            else
                echo "❌ Foundry installation failed"
                exit 1
            fi
        fi

        STEP=$((STEP + 1))
        echo

        # 8. Install and Run Docker
        echo "[$STEP/$TOTAL_STEPS] ----- Installing Docker Engine..."
        if ! command -v docker &> /dev/null; then
            echo "Docker not found, installing..."
            sudo sysctl -w kernel.apparmor_restrict_unprivileged_userns=0
            sudo apt-get install -y gnome-terminal

            # Add Docker's official GPG key:
            sudo apt-get install -y ca-certificates curl
            sudo install -m 0755 -d /etc/apt/keyrings
            sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
            sudo chmod a+r /etc/apt/keyrings/docker.asc

            # Add the repository to Apt sources:
            echo \
              "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
              $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
            # Install the Docker packages.
            sudo apt-get update -y
            sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
        else
            echo "Docker is already installed."
        fi

        # Run Docker Daemon
        echo "Starting Docker Daemon..."
        sudo systemctl start docker
        sudo chmod 666 /var/run/docker.sock

        STEP=$((STEP + 1))
        echo

        SUCCESS="true"
        echo

    # If it is an operating system other than Ubuntu, execute the following commands.
    else
        echo "$OS_NAME is an unsupported operating system."
    fi
fi

# Function to check if a command exists and its version if necessary
function check_command_version {
    CMD=$1
    EXPECTED_VERSION=$2
    VERSION_CMD=$3

    if command -v "$CMD" &> /dev/null; then

        # If zsh, enable word splitting option locally.
        if [[ "$OS_TYPE" == "darwin" ]]; then
            setopt localoptions sh_word_split
        fi

        CURRENT_VERSION=$(eval $VERSION_CMD 2>&1 | head -n 1)

        if [[ -z "$EXPECTED_VERSION" ]]; then
            if [[ "$CMD" == "forge" || "$CMD" == "cast" || "$CMD" == "anvil" ]]; then
                echo "✅ foundry - $CMD is installed. Current version: $CURRENT_VERSION"
            else
                echo "✅ $CMD is installed. Current version: $CURRENT_VERSION"
            fi
        elif echo "$CURRENT_VERSION" | grep -q "$EXPECTED_VERSION"; then
            echo "✅ $CMD is installed and matches version $EXPECTED_VERSION."
        else
            echo "❌ $CMD is installed but version does not match $EXPECTED_VERSION. Current version: $CURRENT_VERSION"
        fi
    else
        if [[ "$CMD" == "forge" || "$CMD" == "cast" || "$CMD" == "anvil" ]]; then
            echo "❌ foundry - $CMD is not installed."
        else
            echo "❌ $CMD is not installed."
        fi
    fi
}

if [[ "$SUCCESS" == "true" ]]; then
    echo "All required tools are installed and ready to use!"
else
    echo "Some tools failed to install. Please check the output above for details."
    exit 1
fi

# Final step: Check installation and versions
echo "Verifying installation and versions..."
if [ "$SHELL_NAME" = "zsh" ]; then
    zsh -c "source $CONFIG_FILE"
elif [ "$SHELL_NAME" = "bash" ]; then
    bash -c "source $CONFIG_FILE"
fi

# Check Homebrew (for MacOS) - Just check if installed
if [[ "$OS_TYPE" == "darwin" ]]; then
    # Check Homebrew
    check_command_version brew "" "brew --version"

    # Check Git
    check_command_version git "" "git --version"

    # Check Make
    check_command_version make "" "make --version"

    # Check Xcode
    check_command_version xcode-select "" "xcode-select --version"

    # Check Go (Expect version 1.22.6)
    check_command_version go "go1.22.6" "go version"

    # Check Node.js (Expect version 20.16.0)
    check_command_version node "v20.16.0" "node -v"

    # Check Pnpm
    check_command_version pnpm "" "pnpm --version"

    # Check Docker
    check_command_version docker "" "docker --version"

    # Check Foundry
    check_command_version forge "" "forge --version"
    check_command_version cast "" "cast --version"
    check_command_version anvil "" "anvil --version"

    echo "🎉 All required tools are installed and ready to use!"


elif [[ "$OS_TYPE" == "linux" ]]; then
    # Check Git
    check_command_version git "" "git --version"

    # Check Make
    check_command_version make "" "make --version"

    # Check gcc (Instead build-essential)
    check_command_version gcc "" "gcc --version"

    # Check Go (Expect version 1.22.6)
    check_command_version go "go1.22.6" "go version"

    # Check Node.js (Expect version 20.16.0)
    check_command_version node "v20.16.0" "node -v"

    # Check Pnpm
    check_command_version pnpm "" "pnpm --version"

    # Check Docker
    check_command_version docker "" "docker --version"

    # Check Foundry
    check_command_version forge "" "forge --version"
    check_command_version cast "" "cast --version"
    check_command_version anvil "" "anvil --version"

    echo "🎉 All required tools are installed and ready to use!"
fi