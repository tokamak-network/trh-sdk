#!/usr/bin/env bash
# Re-run with the correct interpreter depending on the OS
# Use SKIP_SHEBANG_CHECK variable to prevent infinite loop if already re-run
# Get machine architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)
        ARCH="x86_64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

OS_TYPE=$(uname)

TOTAL_MACOS_STEPS=14
TOTAL_LINUX_STEPS=9
STEP=1
SUCCESS="false"

if [ "$OS_TYPE" = "Darwin" ] && [ -z "$SKIP_SHEBANG_CHECK" ]; then
  if [ -x "/bin/zsh" ]; then
    export SKIP_SHEBANG_CHECK=1
    echo "macOS detected. Switching to zsh interpreter......"
    exec /bin/zsh "$0" "$@"
  else
    echo "Error: /bin/zsh not found. Please ensure zsh is installed." >&2
    exit 1
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


if [[ "$OS_TYPE" == "Darwin" ]]; then
    # 1. Install Homebrew
    echo "[$STEP/$TOTAL_MACOS_STEPS] Installing Homebrew..."
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
    echo "[$STEP/$TOTAL_MACOS_STEPS] Installing Git..."
    if ! command -v git &> /dev/null; then
        echo "git not found, installing..."
        brew install git
    else
        echo "git is already installed."
    fi

    STEP=$((STEP + 1))
    echo

    # 3. Install Xcode Command Line Tools(Inclue make)
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
            # shellcheck disable=SC2086
            rm terraform_${TERRAFORM_LATEST_VERSION}_darwin_arm64.zip
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
        brew install kubectl
    else
        echo "kubectl is already installed."
    fi
    STEP=$((STEP + 1))

    # 8. Install Node.js
    echo "[$STEP/$TOTAL_MACOS_STEPS] ----- Installing Node.js (v20.16.0)..."
    # Save the current Node.js version
    current_node_version=$(node -v 2>/dev/null)

    # Check if the current version is not v20.16.0
    if [[ "$current_node_version" != "v20.16.0" ]]; then

        # 5-1. Install NVM
        echo "[$STEP/$TOTAL_MACOS_STEPS] ----- Installing NVM..."

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

        # 5-2. Install Node.js v20.16.0 using NVM
        echo "[$STEP/$TOTAL_MACOS_STEPS] ----- Installing Node.js v20.16.0 using NVM..."
        if ! nvm ls | grep 'v20.16.0' | grep -v 'default' &> /dev/null; then
            echo "Node.js v20.16.0 not found, installing..."
            nvm install v20.16.0
        else
            echo "Node.js v20.16.0 is already installed."
        fi

        # 5-3. Set Node.js v20.16.0 as the default version
        echo "[$STEP/$TOTAL_MACOS_STEPS] ----- Setting Node.js v20.16.0 as the default version..."
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
    echo "[$STEP/$TOTAL_MACOS_STEPS] ----- Installing Pnpm..."
    if ! command -v pnpm &> /dev/null; then
        echo "pnpm not found, installing..."
        brew install pnpm
    else
        echo "pnpm is already installed."
    fi

    STEP=$((STEP + 1))
    echo

    # 7. Install Cargo (v1.83.0)
    echo "[$STEP/$TOTAL_MACOS_STEPS] ----- Installing Cargo (v1.83.0)..."
    source "$HOME/.cargo/env"
    if ! cargo --version | grep "1.83.0" &> /dev/null; then
        echo "Cargo 1.83.0 not found, installing..."
        curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y

        # Check if the Cargo configuration is already in the CONFIG_FILE
        if ! grep -Fq '. "$HOME/.cargo/env"' "$CONFIG_FILE"; then

            # If the configuration is not found, add Cargo to the current shell session
            {
                echo ''
                echo '. "$HOME/.cargo/env"'
            } >> "$CONFIG_FILE"
        fi

        # Check if the Cargo configuration is already in the PROFILE_FILE
        if ! grep -Fq '. "$HOME/.cargo/env"' "$PROFILE_FILE"; then
            # If the configuration is not found, add Cargo to the current shell session
            {
                echo ''
                echo '. "$HOME/.cargo/env"'
            } >> "$PROFILE_FILE"
        fi

        source "$HOME/.cargo/env"
        rustup install 1.83.0
        rustup default 1.83.0
    else
        echo "Cargo 1.83.0 is already installed."
    fi

    STEP=$((STEP + 1))
    echo

    # 8. Install Node.js
    echo "[$STEP/$TOTAL_MACOS_STEPS] ----- Installing Node.js (v20.16.0)..."
    # Save the current Node.js version
    current_node_version=$(node -v 2>/dev/null)

    # Check if the current version is not v20.16.0
    if [[ "$current_node_version" != "v20.16.0" ]]; then

        # 5-1. Install NVM
        echo "[$STEP/$TOTAL_MACOS_STEPS] ----- Installing NVM..."

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

        # 5-2. Install Node.js v20.16.0 using NVM
        echo "[$STEP/$TOTAL_MACOS_STEPS] ----- Installing Node.js v20.16.0 using NVM..."
        if ! nvm ls | grep 'v20.16.0' | grep -v 'default' &> /dev/null; then
            echo "Node.js v20.16.0 not found, installing..."
            nvm install v20.16.0
        else
            echo "Node.js v20.16.0 is already installed."
        fi

        # 5-3. Set Node.js v20.16.0 as the default version
        echo "[$STEP/$TOTAL_MACOS_STEPS] ----- Setting Node.js v20.16.0 as the default version..."
        echo "Switching to Node.js v20.16.0..."
        nvm use v20.16.0
        nvm alias default v20.16.0
        echo "Node.js v20.16.0 is now set as the default version."
    else
        echo "Node.js is already v20.16.0."
    fi

    STEP=$((STEP + 1))
    echo

    # 9. Install Pnpm
    echo "[$STEP/$TOTAL_MACOS_STEPS] ----- Installing Pnpm..."
    if ! command -v pnpm &> /dev/null; then
        echo "pnpm not found, installing..."
        brew install pnpm
    else
        echo "pnpm is already installed."
    fi

    STEP=$((STEP + 1))
    echo

    # 10. Install Cargo (v1.83.0)
    echo "[$STEP/$TOTAL_MACOS_STEPS] ----- Installing Cargo (v1.83.0)..."
    source "$HOME/.cargo/env"
    if ! cargo --version | grep "1.83.0" &> /dev/null; then
        echo "Cargo 1.83.0 not found, installing..."
        curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y

        # Check if the Cargo configuration is already in the CONFIG_FILE
        if ! grep -Fq '. "$HOME/.cargo/env"' "$CONFIG_FILE"; then

            # If the configuration is not found, add Cargo to the current shell session
            {
                echo ''
                echo '. "$HOME/.cargo/env"'
            } >> "$CONFIG_FILE"
        fi

        # Check if the Cargo configuration is already in the PROFILE_FILE
        if ! grep -Fq '. "$HOME/.cargo/env"' "$PROFILE_FILE"; then
            # If the configuration is not found, add Cargo to the current shell session
            {
                echo ''
                echo '. "$HOME/.cargo/env"'
            } >> "$PROFILE_FILE"
        fi

        source "$HOME/.cargo/env"
        rustup install 1.83.0
        rustup default 1.83.0
    else
        echo "Cargo 1.83.0 is already installed."
    fi

    STEP=$((STEP + 1))
    echo
    # 11. Install Foundry
    echo "[$STEP/$TOTAL_MACOS_STEPS] ----- Installing Foundry..."
    echo "Installing Foundry..."
    if ! command -v jq &> /dev/null; then
        echo "jq not found, installing..."
        brew install jq
    else
        echo "jq is already installed."
    fi
  
    # Check if Foundry is already installed with expected version
    if forge --version &> /dev/null && cast --version &> /dev/null; then
        echo "✅ Foundry is already installed"
    else
        # Install Foundry
        echo "Installing/updating Foundry..."
        if ! command -v curl &> /dev/null; then
            echo "curl not found, installing..."
            brew install curl
        fi
        if curl -L https://foundry.paradigm.xyz | bash && curl -fsSL https://raw.githubusercontent.com/tokamak-network/trh-sdk/main/scripts/install-foundry.sh | bash; then \
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

elif [[ "$OS_TYPE" == "Linux" ]]; then
    if ! command -v sudo &> /dev/null; then
        echo "sudo not found, installing..."
        apt-get install -y sudo
    else
        echo "sudo is already installed."
    fi

    # 1. Install Build-essential
    echo "[$STEP/$TOTAL_LINUX_STEPS] ----- Installing Build-essential..."
    if ! dpkg -s build-essential &> /dev/null; then
        echo "Build-essential not found, installing..."
        sudo apt-get install -y build-essential
    else
        echo "Build-essential is already installed."
    fi

    # Check if jq is installed
    if ! command -v jq &> /dev/null; then
        echo "jq not found, installing..."
        sudo apt-get install -y jq
    else
        echo "✅ jq is already installed"
    fi

    STEP=$((STEP + 1))
    echo

    # 2. Install Terraform
    echo "[$STEP/$TOTAL_LINUX_STEPS] ----- Installing Terraform..."

    # Check if Terraform is already installed with version 1.1+
    if command -v terraform &> /dev/null && current_version=$(terraform --version | grep -oP "v\K[0-9]+\.[0-9]+") && (( $(echo "$current_version >= 1.1" | bc -l) )); then
        echo "✅ Terraform v$current_version is already installed"
    else
        echo "Terraform not found, installing..."

        sudo apt-get update && sudo apt-get install -y gnupg software-properties-common curl
        curl -fsSL https://apt.releases.hashicorp.com/gpg | gpg --dearmor | sudo tee /usr/share/keyrings/hashicorp-archive-keyring.gpg > /dev/null

        echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
        sudo apt-get update && sudo apt-get install -y terraform


        # Verify installation
        if command -v terraform &> /dev/null; then
            echo "✅ Terraform has been installed successfully!"
            terraform --version
        else
            echo "❌ Terraform installation failed"
            exit 1
        fi
    fi
    
    STEP=$((STEP + 1))
    echo


    # Step 3: Install AWS CLI
    echo "[$STEP/$TOTAL_LINUX_STEPS] ----- Installing AWS CLI..."
    # Check if AWS CLI v2 is already installed
    if command -v aws &> /dev/null && version=$(aws --version | cut -d/ -f2 | cut -d' ' -f1) && [[ $version == 2* ]]; then
        echo "✅ AWS CLI v2 is already installed (version $version)"
    else 
        echo "Installing AWS CLI v2..."

        # Install unzip if not present
        if ! command -v unzip &> /dev/null; then
            echo "Installing unzip..."
            sudo apt-get update && sudo apt-get install -y unzip
        fi

        # Download AWS CLI v2 installer based on architecture
        if [ "$ARCH" = "x86_64" ]; then
            curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
        elif [ "$ARCH" = "arm64" ]; then
            curl "https://awscli.amazonaws.com/awscli-exe-linux-aarch64.zip" -o "awscliv2.zip"
        else
            echo "❌ Unsupported architecture: $ARCH"
            exit 1
        fi

        # Unzip the installer
        unzip awscliv2.zip

        # Install AWS CLI v2
        sudo ./aws/install --bin-dir /usr/local/bin --install-dir /usr/local/aws-cli --update

        # Clean up downloaded files
        rm -rf aws awscliv2.zip

        # Verify installation
        if command -v aws &> /dev/null; then
            echo "✅ AWS CLI v2 has been installed successfully!"
            aws --version
        else
            echo "❌ AWS CLI v2 installation failed"
            exit 1
        fi
    fi

    STEP=$((STEP + 1))
    echo


    # Step 4: Install Helm
    echo "[$STEP/$TOTAL_LINUX_STEPS] ----- Installing Helm..."
    # Check if Helm is already installed
    if command -v helm &> /dev/null; then
        echo "✅ Helm is already installed"
        helm version
    else 
        echo "Installing Helm..."

        # Download Helm installer script
        curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
        
        # Make installer executable
        chmod 700 get_helm.sh
        
        # Run installer
        ./get_helm.sh

        # Clean up installer
        rm get_helm.sh

        # Verify installation
        if command -v helm &> /dev/null; then
            echo "✅ Helm has been installed successfully!"
            helm version
        else
            echo "❌ Helm installation failed"
            exit 1
        fi
    fi
    
    STEP=$((STEP + 1))
    echo


    # Step 5: Install kubectl
    echo "[$STEP/$TOTAL_LINUX_STEPS] ----- Installing kubectl..."
    # Check if kubectl is already installed
    if command -v kubectl &> /dev/null; then
        echo "✅ kubectl is already installed"
        kubectl version --client
    else
        echo "Installing kubectl..."
        # Download latest kubectl binary based on architecture
        if [[ "$ARCH" == "arm64" ]]; then
            curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/arm64/kubectl"
            curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/arm64/kubectl.sha256"
        else
            curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
            curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl.sha256"
        fi

        # Validate binary against checksum
        if echo "$(cat kubectl.sha256)  kubectl" | sha256sum --check; then
            echo "kubectl checksum validation: OK"
        else
            echo "❌ kubectl checksum validation failed"
            rm kubectl kubectl.sha256
            exit 1
        fi

        # Install kubectl
        sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

        # Clean up downloaded files
        rm kubectl kubectl.sha256

        # Verify installation
        if command -v kubectl &> /dev/null; then
            echo "✅ kubectl has been installed successfully!"
            kubectl version --client
        else
            echo "❌ kubectl installation failed"
            exit 1
        fi
    fi

    STEP=$((STEP + 1))
    echo


    # Step 6: Install Node.js
    echo "[$STEP/$TOTAL_LINUX_STEPS] ----- Installing Node.js (v20.16.0)..."

    # Save the current Node.js version
    current_node_version=$(node -v 2>/dev/null)

    # Check if the current version is not v20.16.0
    if [[ "$current_node_version" != "v20.16.0" ]]; then

        # 5-1. Install NVM
        echo "[$STEP/$TOTAL_LINUX_STEPS] ----- Installing NVM..."

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
        echo "[$STEP/$TOTAL_LINUX_STEPS] ----- Installing Node.js v20.16.0 using NVM..."
        if ! nvm ls | grep 'v20.16.0' | grep -v 'default' &> /dev/null; then
            echo "Node.js v20.16.0 not found, installing..."
            nvm install v20.16.0
        else
            echo "Node.js v20.16.0 is already installed."
        fi

        # 5-3. Set Node.js v20.16.0 as the default version
        echo "[$STEP/$TOTAL_LINUX_STEPS] ----- Setting Node.js v20.16.0 as the default version..."
        echo "Switching to Node.js v20.16.0..."
        nvm use v20.16.0
        nvm alias default v20.16.0
        echo "Node.js v20.16.0 is now set as the default version."
    else
        echo "Node.js is already v20.16.0."
    fi

    STEP=$((STEP + 1))
    echo

    # 7. Install Pnpm
    echo "[$STEP/$TOTAL_LINUX_STEPS] ----- Installing Pnpm..."
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

    # 7. Install Cargo (v1.83.0)
    echo "[$STEP/$TOTAL_LINUX_STEPS] ----- Installing Cargo (v1.83.0)..."
    source "$HOME/.cargo/env"
    if ! cargo --version | grep -q "1.83.0" &> /dev/null; then
        echo "Cargo 1.83.0 not found, installing..."
        curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y

        # Check if the Cargo configuration is already in the CONFIG_FILE
        if ! grep -Fq '. "$HOME/.cargo/env"' "$CONFIG_FILE"; then

            # If the configuration is not found, add Cargo to the current shell session
            {
                echo ''
                echo '. "$HOME/.cargo/env"'
            } >> "$CONFIG_FILE"
        fi

        # Check if the Cargo configuration is already in the PROFILE_FILE
        if ! grep -Fq '. "$HOME/.cargo/env"' "$PROFILE_FILE"; then
            # If the configuration is not found, add Cargo to the current shell session
            {
                echo ''
                echo '. "$HOME/.cargo/env"'
            } >> "$PROFILE_FILE"
        fi

        source "$HOME/.cargo/env"
        rustup install 1.83.0
        rustup default 1.83.0
    else
        echo "Cargo 1.83.0 is already installed."
    fi

    STEP=$((STEP + 1))
    echo

    # 9. Install Foundry
    echo "[$STEP/$TOTAL_LINUX_STEPS] ----- Installing Foundry..."
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
        if curl -L https://foundry.paradigm.xyz | bash && curl -fsSL https://raw.githubusercontent.com/tokamak-network/trh-sdk/main/scripts/install-foundry.sh | bash; then \
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


# Check Homebrew (for MacOS) - Just check if installed
if [[ "$OS_TYPE" == "Darwin" ]]; then
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

    # Check Cargo (Expect version 1.83.0)
    check_command_version cargo "1.83.0" "cargo --version"

    # Check Foundry
    check_command_version forge "" "forge --version"
    check_command_version cast "" "cast --version"
    check_command_version anvil "" "anvil --version"
    check_command_version kubectl "" "kubectl version --client"
    check_command_version helm "" "helm version"
    check_command_version terraform "" "terraform --version"
    check_command_version aws "" "aws --version"

    echo "🎉 All required tools are installed and ready to use!"


elif [[ "$OS_TYPE" == "Linux" ]]; then
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

    # Check Cargo (Expect version 1.83.0

    # Check Foundry
    check_command_version forge "" "forge --version"
    check_command_version cast "" "cast --version"
    check_command_version anvil "" "anvil --version"

    check_command_version kubectl "" "kubectl version --client"
    check_command_version helm "" "helm version"
    check_command_version terraform "" "terraform --version"
    check_command_version aws "" "aws --version"

    echo "🎉 All required tools are installed and ready to use!"
fi