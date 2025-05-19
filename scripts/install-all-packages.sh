#!/usr/bin/env bash

# Re-run with the correct interpreter depending on the OS
# Use SKIP_SHEBANG_CHECK variable to prevent infinite loop if already re-run
# Get machine architecture
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

OS_TYPE=$(uname)

TOTAL_MACOS_STEPS=15
TOTAL_LINUX_STEPS=12
STEP=1
SUCCESS="false"

# Detect current shell
CURRENT_SHELL=$(ps -p $$ -o comm=)

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

# Function to display completion message
function display_completion_message {
    if [[ "$SUCCESS" == "true" ]]; then
        echo ""
        echo "All steps are complete."
        echo ""
        exit 0
    else
        echo ""
        echo "Installation was interrupted. Completed $((STEP - 1)) steps."
        echo ""
        echo "Please source your profile to apply changes:"
        echo -e "\033[1;32msource $CONFIG_FILE\033[0m"
        exit 1
    fi
}

# Use trap to display message on script exit, whether successful or due to an error
trap display_completion_message EXIT
trap "echo 'Process interrupted!'; exit 1" INT

if [[ "$OS_TYPE" == "Darwin" ]]; then
    # 1. Install Homebrew
    echo "[$STEP/$TOTAL_MACOS_STEPS] Installing Homebrew..."
    if ! command -v brew &> /dev/null; then
        echo "Homebrew not found, installing..."
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

        if [[ "$ARCH" == "aarch64" || "$ARCH" == "arm64" ]]; then
            # Check if the Homebrew configuration is already in the CONFIG_FILE
            if ! grep -Fxq 'export PATH="/usr/local/bin:$PATH"' "$CONFIG_FILE"; then
                {
                    echo ''
                    echo 'export PATH="/usr/local/bin:$PATH"'
                } >> "$CONFIG_FILE"
            fi

            # Check if the Homebrew configuration is already in the PROFILE_FILE
            if ! grep -Fxq 'export PATH="/usr/local/bin:$PATH"' "$PROFILE_FILE"; then
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
    echo "[$STEP/$TOTAL_MACOS_STEPS] Installing Git..."
    if ! command -v git &> /dev/null; then
        echo "git not found, installing..."
        brew install git
    else
        echo "git is already installed."
    fi
    STEP=$((STEP + 1))
    echo

    # 3. Install Xcode Command Line Tools
    echo "[$STEP/$TOTAL_MACOS_STEPS] Installing Xcode Command Line Tools..."
    if ! xcode-select -p &> /dev/null; then
        echo "Xcode Command Line Tools not found, installing..."
        xcode-select --install
    else
        echo "Xcode Command Line Tools are already installed. Checking for updates..."
    fi
    STEP=$((STEP + 1))
    echo

    # 4. Install terraform
    echo "[$STEP/$TOTAL_MACOS_STEPS] Installing Terraform..."
    TERRAFORM_LATEST_VERSION=$(curl -s https://api.github.com/repos/hashicorp/terraform/releases/latest | jq -r '.tag_name' | sed 's/^v//')
    if command -v terraform &> /dev/null; then
        current_version=$(terraform --version | sed -nE 's/^Terraform v([0-9]+\.[0-9]+).*/\1/p')
        if (( $(echo "$current_version >= 1.11" | bc -l) )); then
            echo "Terraform v$current_version is already installed and meets the version requirement."
        else
            echo "Terraform v$current_version is installed but does not meet the version requirement. Updating..."
            if [[ "$ARCH" == "arm64" ]]; then
                curl -LO "https://releases.hashicorp.com/terraform/${TERRAFORM_LATEST_VERSION}/terraform_${TERRAFORM_LATEST_VERSION}_darwin_arm64.zip"
                unzip terraform_"${TERRAFORM_LATEST_VERSION}"_darwin_arm64.zip
                sudo mv terraform /usr/local/bin/
                rm terraform_"${TERRAFORM_LATEST_VERSION}"_darwin_arm64.zip
            else
                curl -LO "https://releases.hashicorp.com/terraform/${TERRAFORM_LATEST_VERSION}/terraform_${TERRAFORM_LATEST_VERSION}_darwin_amd64.zip"
                unzip terraform_"${TERRAFORM_LATEST_VERSION}"_darwin_amd64.zip
                sudo mv terraform /usr/local/bin/
                rm terraform_"${TERRAFORM_LATEST_VERSION}"_darwin_amd64.zip
            fi
        fi
    else
        echo "Terraform not found, installing..."
        if [[ "$ARCH" == "arm64" ]]; then
            curl -LO "https://releases.hashicorp.com/terraform/${TERRAFORM_LATEST_VERSION}/terraform_${TERRAFORM_LATEST_VERSION}_darwin_arm64.zip"
            unzip terraform_"${TERRAFORM_LATEST_VERSION}"_darwin_arm64.zip
            sudo mv terraform /usr/local/bin/
            rm terraform_"${TERRAFORM_LATEST_VERSION}"_darwin_arm64.zip
        else
            curl -LO "https://releases.hashicorp.com/terraform/${TERRAFORM_LATEST_VERSION}/terraform_${TERRAFORM_LATEST_VERSION}_darwin_amd64.zip"
            unzip terraform_"${TERRAFORM_LATEST_VERSION}"_darwin_amd64.zip
            sudo mv terraform /usr/local/bin/
            rm terraform_"${TERRAFORM_LATEST_VERSION}"_darwin_amd64.zip
        fi
    fi
    STEP=$((STEP + 1))
    echo

    # 5. Install AWS CLI
    echo "[$STEP/$TOTAL_MACOS_STEPS] Installing AWS CLI..."
    if command -v aws &> /dev/null; then
        current_version=$(aws --version | cut -d/ -f2 | cut -d' ' -f1)
        if [[ $current_version == 2* ]]; then
            echo "AWS CLI v$current_version is already installed and meets the version requirement."
        else
            echo "AWS CLI v$current_version is installed but does not meet the version requirement. Updating..."
            brew upgrade awscli
        fi
    else
        echo "AWS CLI not found, installing..."
        brew install awscli
    fi
    STEP=$((STEP + 1))
    echo

    # 6. Install Helm
    echo "[$STEP/$TOTAL_MACOS_STEPS] Installing Helm..."
    if ! command -v helm &> /dev/null; then
        echo "Helm not found, installing..."
        brew install helm
    else
        echo "Helm is already installed."
    fi
    STEP=$((STEP + 1))
    echo

    # 7. Install kubectl
    echo "[$STEP/$TOTAL_MACOS_STEPS] Installing kubectl..."
    if ! command -v kubectl &> /dev/null; then
        echo "kubectl not found, installing..."
        if [[ "$ARCH" == "arm64" ]]; then
            curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/darwin/arm64/kubectl"
            curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/darwin/arm64/kubectl.sha256"
            chmod +x ./kubectl
            sudo mv ./kubectl /usr/local/bin/kubectl
            sudo chown root: /usr/local/bin/kubectl
        else
            curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/darwin/amd64/kubectl"
            curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/darwin/amd64/kubectl.sha256"
            chmod +x ./kubectl
            sudo mv ./kubectl /usr/local/bin/kubectl
            sudo chown root: /usr/local/bin/kubectl
        fi 
    else
        echo "kubectl is already installed."
    fi
    STEP=$((STEP + 1))
    echo

    # 8. Install Node.js
    echo "[$STEP/$TOTAL_MACOS_STEPS] Installing Node.js (v20.16.0)..."
    current_node_version=$(node -v 2>/dev/null)
    if [[ "$current_node_version" != "v20.16.0" ]]; then
        # Install NVM
        echo "Installing NVM..."
        export NVM_DIR="$HOME/.nvm"
        mkdir -p "$NVM_DIR"
        HOMEBREW_PREFIX=$(brew --prefix)
        [ -s "$HOMEBREW_PREFIX/opt/nvm/nvm.sh" ] && \. "$HOMEBREW_PREFIX/opt/nvm/nvm.sh"
        [ -s "$HOMEBREW_PREFIX/opt/nvm/etc/bash_completion.d/nvm" ] && \. "$HOMEBREW_PREFIX/opt/nvm/etc/bash_completion.d/nvm"

        if ! command -v nvm &> /dev/null; then
            brew install nvm
            if ! grep -Fxq 'export NVM_DIR="$HOME/.nvm"' "$CONFIG_FILE"; then
                {
                    echo ''
                    echo 'export NVM_DIR="$HOME/.nvm"'
                    echo "[ -s \"$HOMEBREW_PREFIX/opt/nvm/nvm.sh\" ] && \. \"$HOMEBREW_PREFIX/opt/nvm/nvm.sh\""
                    echo "[ -s \"$HOMEBREW_PREFIX/opt/nvm/etc/bash_completion.d/nvm\" ] && \. \"$HOMEBREW_PREFIX/opt/nvm/etc/bash_completion.d/nvm\""
                } >> "$CONFIG_FILE"
            fi
            if ! grep -Fxq 'export NVM_DIR="$HOME/.nvm"' "$PROFILE_FILE"; then
                {
                    echo ''
                    echo 'export NVM_DIR="$HOME/.nvm"'
                    echo "[ -s \"$HOMEBREW_PREFIX/opt/nvm/nvm.sh\" ] && \. \"$HOMEBREW_PREFIX/opt/nvm/nvm.sh\""
                    echo "[ -s \"$HOMEBREW_PREFIX/opt/nvm/etc/bash_completion.d/nvm\" ] && \. \"$HOMEBREW_PREFIX/opt/nvm/etc/bash_completion.d/nvm\""
                } >> "$PROFILE_FILE"
            fi
        fi

        # Install Node.js v20.16.0
        nvm install v20.16.0
        nvm use v20.16.0
        nvm alias default v20.16.0
    else
        echo "Node.js v20.16.0 is already installed."
    fi
    STEP=$((STEP + 1))
    echo

    # 9. Install Pnpm
    echo "[$STEP/$TOTAL_MACOS_STEPS] Installing Pnpm..."
    if ! command -v pnpm &> /dev/null; then
        echo "pnpm not found, installing..."
        brew install pnpm
    else
        echo "pnpm is already installed."
    fi
    STEP=$((STEP + 1))
    echo

    # 11. Install Foundry
    echo "[$STEP/$TOTAL_MACOS_STEPS] Installing Foundry..."
    if ! command -v jq &> /dev/null; then
        echo "jq not found, installing..."
        brew install jq
    fi

    if forge --version &> /dev/null && cast --version &> /dev/null; then
        echo "Foundry is already installed"
    else
        echo "Installing/updating Foundry..."
        if ! command -v curl &> /dev/null; then
            brew install curl
        fi
        if curl -L https://foundry.paradigm.xyz | bash && curl -fsSL https://raw.githubusercontent.com/tokamak-network/trh-sdk/main/scripts/install-foundry.sh | bash; then
            export PATH="$HOME/.foundry/bin:$PATH"
            echo "Foundry has been installed successfully!"
        else
            echo "Foundry installation failed"
            exit 1
        fi
    fi
    STEP=$((STEP + 1))
    echo

    # 12. Install Docker
    echo "[$STEP/$TOTAL_MACOS_STEPS] Installing Docker..."
    if ! command -v docker &> /dev/null; then
        echo "Docker not found, installing..."
        brew install --cask docker
    else
        echo "Docker is already installed."
    fi

    # Run Docker Daemon
    echo "Starting Docker Daemon..."
    if ! docker info > /dev/null 2>&1; then
        echo "🚫 Docker is not running. Starting Docker Desktop..."
        open -a Docker

        # Wait for Docker to initialize
        while ! docker info > /dev/null 2>&1; do
            echo "⏳ Waiting for Docker to start..."
            sleep 2
        done

        echo "✅ Docker is now running!"
    else
        echo "✅ Docker is already running."
    fi
    STEP=$((STEP + 1))
    echo

    SUCCESS="true"

elif [[ "$OS_TYPE" == "Linux" ]]; then
    if ! command -v sudo &> /dev/null; then
        echo "sudo not found, installing..."
        apt-get install -y sudo
    fi

    # 1. Update package list
    echo "[$STEP/$TOTAL_LINUX_STEPS] Updating package list..."
    sudo apt-get update -y
    STEP=$((STEP + 1))
    echo

    # 2. Install Build-essential
    echo "[$STEP/$TOTAL_LINUX_STEPS] Installing Build-essential..."
    if ! dpkg -s build-essential &> /dev/null; then
        echo "Build-essential not found, installing..."
        sudo apt-get install -y build-essential
    else
        echo "Build-essential is already installed."
    fi
    STEP=$((STEP + 1))
    echo

    # 3. Install Git
    echo "[$STEP/$TOTAL_LINUX_STEPS] Installing Git..."
    if ! command -v git &> /dev/null; then
        echo "git not found, installing..."
        sudo apt-get install -y git
    else
        echo "git is already installed."
    fi
    STEP=$((STEP + 1))
    echo

    # 4. Install Terraform
    echo "[$STEP/$TOTAL_LINUX_STEPS] Installing Terraform..."
    if command -v terraform &> /dev/null && current_version=$(terraform --version | grep -oP "v\K[0-9]+\.[0-9]+") && (( $(echo "$current_version >= 1.1" | bc -l) )); then
        echo "Terraform v$current_version is already installed"
    else
        echo "Installing Terraform..."
        sudo apt-get install -y gnupg software-properties-common curl
        curl -fsSL https://apt.releases.hashicorp.com/gpg | gpg --dearmor | sudo tee /usr/share/keyrings/hashicorp-archive-keyring.gpg > /dev/null
        echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
        sudo apt-get update && sudo apt-get install -y terraform
    fi
    STEP=$((STEP + 1))
    echo

    # 5. Install AWS CLI
    echo "[$STEP/$TOTAL_LINUX_STEPS] Installing AWS CLI..."
    if command -v aws &> /dev/null && version=$(aws --version | cut -d/ -f2 | cut -d' ' -f1) && [[ $version == 2* ]]; then
        echo "AWS CLI v2 is already installed (version $version)"
    else
        echo "Installing AWS CLI v2..."
        if ! command -v unzip &> /dev/null; then
            sudo apt-get install -y unzip
        fi
        if [ "$ARCH" = "arm64" ]; then
            curl "https://awscli.amazonaws.com/awscli-exe-linux-aarch64.zip" -o "awscliv2.zip"
        else
            curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
        fi
        unzip awscliv2.zip
        sudo ./aws/install --bin-dir /usr/local/bin --install-dir /usr/local/aws-cli --update
        rm -rf aws awscliv2.zip
    fi
    STEP=$((STEP + 1))
    echo

    # 6. Install Helm
    echo "[$STEP/$TOTAL_LINUX_STEPS] Installing Helm..."
    if command -v helm &> /dev/null; then
        echo "Helm is already installed"
    else
        echo "Installing Helm..."
        curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
        chmod 700 get_helm.sh
        ./get_helm.sh
        rm get_helm.sh
    fi
    STEP=$((STEP + 1))
    echo

    # 7. Install kubectl
    echo "[$STEP/$TOTAL_LINUX_STEPS] Installing kubectl..."
    if command -v kubectl &> /dev/null; then
        echo "kubectl is already installed"
    else
        echo "Installing kubectl..."
        if [[ "$ARCH" == "arm64" ]]; then
            curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/arm64/kubectl"
            curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/arm64/kubectl.sha256"
        else
            curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
            curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl.sha256"
        fi
        if echo "$(cat kubectl.sha256)  kubectl" | sha256sum --check; then
            sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
            rm kubectl kubectl.sha256
        else
            echo "kubectl checksum validation failed"
            rm kubectl kubectl.sha256
            exit 1
        fi
    fi
    STEP=$((STEP + 1))
    echo

    # 8. Install Node.js
    echo "[$STEP/$TOTAL_LINUX_STEPS] Installing Node.js (v20.16.0)..."
    current_node_version=$(node -v 2>/dev/null)
    if [[ "$current_node_version" != "v20.16.0" ]]; then
        export NVM_DIR="$HOME/.nvm"
        mkdir -p "$NVM_DIR"
        if ! command -v nvm &> /dev/null; then
            echo "Installing NVM..."
            curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash
            if ! grep -Fxq 'export NVM_DIR="$HOME/.nvm"' "$CONFIG_FILE"; then
                {
                    echo ''
                    echo 'export NVM_DIR="$HOME/.nvm"'
                    echo "[ -s \"$NVM_DIR/nvm.sh\" ] && \. \"$NVM_DIR/nvm.sh\""
                    echo "[ -s \"$NVM_DIR/bash_completion\" ] && \. \"$NVM_DIR/bash_completion\""
                } >> "$CONFIG_FILE"
            fi
            if ! grep -Fxq 'export NVM_DIR="$HOME/.nvm"' "$PROFILE_FILE"; then
                {
                    echo ''
                    echo 'export NVM_DIR="$HOME/.nvm"'
                    echo "[ -s \"$NVM_DIR/nvm.sh\" ] && \. \"$NVM_DIR/nvm.sh\""
                    echo "[ -s \"$NVM_DIR/bash_completion\" ] && \. \"$NVM_DIR/bash_completion\""
                } >> "$PROFILE_FILE"
            fi
        fi
        [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
        nvm install v20.16.0
        nvm use v20.16.0
        nvm alias default v20.16.0
    else
        echo "Node.js v20.16.0 is already installed."
    fi
    STEP=$((STEP + 1))
    echo

    # 9. Install Pnpm
    echo "[$STEP/$TOTAL_LINUX_STEPS] Installing Pnpm..."
    export PATH="$HOME/.local/share/pnpm:$PATH"
    if ! command -v pnpm &> /dev/null; then
        echo "Installing pnpm..."
        curl -fsSL https://get.pnpm.io/install.sh | bash -
        if ! grep -Fq 'export PATH="$HOME/.local/share/pnpm:$PATH"' "$CONFIG_FILE"; then
            {
                echo ''
                echo 'export PATH="$HOME/.local/share/pnpm:$PATH"'
            } >> "$CONFIG_FILE"
        fi
        if ! grep -Fq 'export PATH="$HOME/.local/share/pnpm:$PATH"' "$PROFILE_FILE"; then
            {
                echo ''
                echo 'export PATH="$HOME/.local/share/pnpm:$PATH"'
            } >> "$PROFILE_FILE"
        fi
    else
        echo "pnpm is already installed."
    fi
    STEP=$((STEP + 1))
    echo

    # 11. Install Foundry
    echo "[$STEP/$TOTAL_LINUX_STEPS] Installing Foundry..."
    if ! command -v jq &> /dev/null; then
        sudo apt-get install -y jq
    fi
    if forge --version &> /dev/null && cast --version &> /dev/null; then
        echo "Foundry is already installed"
    else
        echo "Installing Foundry..."
        if ! command -v curl &> /dev/null; then
            sudo apt-get install -y curl
        fi
        if curl -L https://foundry.paradigm.xyz | bash && curl -fsSL https://raw.githubusercontent.com/tokamak-network/trh-sdk/main/scripts/install-foundry.sh | bash; then
            export PATH="$HOME/.foundry/bin:$PATH"
            echo "Foundry has been installed successfully!"
        else
            echo "Foundry installation failed"
            exit 1
        fi
    fi
    STEP=$((STEP + 1))
    echo

    # 12. Install Docker
    echo "[$STEP/$TOTAL_LINUX_STEPS] Installing Docker..."
    if ! command -v docker &> /dev/null; then
        echo "Installing Docker..."
        sudo sysctl -w kernel.apparmor_restrict_unprivileged_userns=0
        sudo apt-get install -y gnome-terminal ca-certificates curl
        sudo install -m 0755 -d /etc/apt/keyrings
        sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
        sudo chmod a+r /etc/apt/keyrings/docker.asc
        echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
        sudo apt-get update -y
        sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    else
        echo "Docker is already installed."
    fi

    # Run Docker Daemon
    echo "Starting Docker Daemon..."
    if ! docker info > /dev/null 2>&1; then
        echo "Docker is not running. Starting Docker service..."
        sudo systemctl start docker
        # Wait for Docker to be fully started
        sudo chmod 666 /var/run/docker.sock
        sleep 5
    else
        echo "Docker is already running."
    fi
    STEP=$((STEP + 1))
    echo

    SUCCESS="true"
else
    echo "Unsupported OS: $OS_TYPE"
    exit 1
fi

# Function to check if a command exists and its version if necessary
function check_command_version {
    CMD=$1
    EXPECTED_VERSION=$2
    VERSION_CMD=$3

    if command -v "$CMD" &> /dev/null; then
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


# Check all installed tools
if [[ "$OS_TYPE" == "Darwin" ]]; then
    check_command_version brew "" "brew --version"
    check_command_version git "" "git --version"
    check_command_version make "" "make --version"
    check_command_version xcode-select "" "xcode-select --version"
    check_command_version node "v20.16.0" "node -v"
    check_command_version pnpm "" "pnpm --version"
    check_command_version docker "" "docker --version"
    check_command_version terraform "" "terraform --version"
    check_command_version aws "" "aws --version"
    check_command_version helm "" "helm version"
    check_command_version kubectl "" "kubectl version --client"
    check_command_version forge "" "forge --version"
    check_command_version cast "" "cast --version"
    check_command_version anvil "" "anvil --version"
else
    check_command_version git "" "git --version"
    check_command_version make "" "make --version"
    check_command_version gcc "" "gcc --version"
    check_command_version node "v20.16.0" "node -v"
    check_command_version pnpm "" "pnpm --version"
    check_command_version docker "" "docker --version"
    check_command_version terraform "" "terraform --version"
    check_command_version aws "" "aws --version"
    check_command_version helm "" "helm version"
    check_command_version kubectl "" "kubectl version --client"
    check_command_version forge "" "forge --version"
    check_command_version cast "" "cast --version"
    check_command_version anvil "" "anvil --version"
fi

echo "🎉 All required tools are installed and ready to use!" 