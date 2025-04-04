#!/usr/bin/env bash

TOTAL_STEPS=7
STEP=1

install_terraform() {
    echo "[$STEP/$TOTAL_STEPS] ----- Installing Terraform..."
    # Check if Terraform is already installed with version 1.1+
    if command -v terraform &> /dev/null; then
        current_version=$(terraform --version | grep -oP "v\K[0-9]+\.[0-9]+")
        if (( $(echo "$current_version >= 1.1" | bc -l) )); then
            echo "✅ Terraform v$current_version is already installed"
            return
        fi
    fi

    echo "Installing Terraform..."

    # Add HashiCorp GPG key
    wget -O - https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg

    # Add HashiCorp repository
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list

    # Update package list and install Terraform
    sudo apt update && sudo apt install -y terraform

    # Verify installation
    if command -v terraform &> /dev/null; then
        echo "✅ Terraform has been installed successfully!"
        terraform --version
    else
        echo "❌ Terraform installation failed"
        exit 1
    fi
}

install_aws_cli() {
    STEP=$((STEP + 1))
    echo "[$STEP/$TOTAL_STEPS] ----- Installing AWS CLI..."
    # Check if AWS CLI v2 is already installed
    if command -v aws &> /dev/null; then
        version=$(aws --version | cut -d/ -f2 | cut -d' ' -f1)
        if [[ $version == 2* ]]; then
            echo "✅ AWS CLI v2 is already installed (version $version)"
            return
        fi
    fi

    echo "Installing AWS CLI v2..."

    # Install unzip if not present
    if ! command -v unzip &> /dev/null; then
        echo "Installing unzip..."
        sudo apt-get update && sudo apt-get install -y unzip
    fi

    # Download AWS CLI v2 installer
    curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"

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
}

install_helm() {
    STEP=$((STEP + 1))
    echo "[$STEP/$TOTAL_STEPS] ----- Installing Helm..."
    # Check if Helm is already installed
    if command -v helm &> /dev/null; then
        echo "✅ Helm is already installed"
        helm version
        return
    fi

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
}

install_kubectl() {
    STEP=$((STEP + 1))
    echo "[$STEP/$TOTAL_STEPS] ----- Installing kubectl..."
    # Check if kubectl is already installed
    if command -v kubectl &> /dev/null; then
        echo "✅ kubectl is already installed"
        kubectl version --client
        return
    fi

    echo "Installing kubectl..."

    # Download latest kubectl binary
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"

    # Download kubectl checksum file
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl.sha256"

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
}

install_nodejs() {
    STEP=$((STEP + 1))
    echo "[$STEP/$TOTAL_STEPS] ----- Installing Node.js..."
    # Check if Node.js is already installed and get version
    current_node_version=$(node -v 2>/dev/null)
    if command -v node &> /dev/null; then
        current_node_version=$(node --version)
    fi

    if [[ "$current_node_version" != "v20.16.0" ]]; then
        echo "Installing NVM..."

        export NVM_DIR="$HOME/.nvm"
        mkdir -p "$NVM_DIR"
        [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
        [ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"

        if ! command -v nvm &> /dev/null; then
            echo "NVM not found, installing..."
            sudo curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash

            [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
            [ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"
        else
            echo "NVM is already installed."
        fi

        echo "Installing Node.js v20.16.0 using NVM..."
        if ! nvm ls | grep 'v20.16.0' | grep -v 'default' &> /dev/null; then
            echo "Node.js v20.16.0 not found, installing..."
            nvm install v20.16.0
        else
            echo "Node.js v20.16.0 is already installed."
        fi

        echo "Setting Node.js v20.16.0 as the default version..."
        echo "Switching to Node.js v20.16.0..."
        nvm use v20.16.0
        nvm alias default v20.16.0
        echo "Node.js v20.16.0 is now set as the default version."
        # Verify installation
        if command -v node &> /dev/null; then
            current_version=$(node -v)
            if [[ "$current_version" == "v20.16.0" ]]; then
                echo "✅ Node.js v20.16.0 has been installed successfully!"
                node -v
            else
                echo "❌ Node.js installation failed - wrong version $current_version"
                exit 1
            fi
        else
            echo "❌ Node.js installation failed"
            exit 1
        fi
    else
        echo "Node.js is already v20.16.0."
    fi
}

install_pnpm() {
    STEP=$((STEP + 1))
    echo "[$STEP/$TOTAL_STEPS] ----- Installing pnpm..."
    echo "Installing Pnpm..."
    if ! command -v pnpm &> /dev/null; then
        echo "pnpm not found, installing..."
        curl -fsSL https://get.pnpm.io/install.sh | bash -
        # Verify installation
        if command -v pnpm &> /dev/null; then
            echo "✅ pnpm has been installed successfully!"
            pnpm --version
        else
            echo "❌ pnpm installation failed"
            exit 1
        fi

    else
        echo "✅ pnpm is already installed."
    fi
}
install_build_essential() {
    STEP=$((STEP + 1))
    echo "[$STEP/$TOTAL_STEPS] ----- Installing Build-essential..."
    if ! dpkg -s build-essential &> /dev/null; then
        echo "Build-essential not found, installing..."
        sudo apt-get install -y build-essential
    else
        echo "Build-essential is already installed."
    fi
}

install_cargo() {
    STEP=$((STEP + 1))
    echo "[$STEP/$TOTAL_STEPS] ----- Installing Cargo (v1.83.0)..."
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
}


install_foundry() {
    STEP=$((STEP + 1))
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
    if command -v forge &> /dev/null && command -v cast &> /dev/null && command -v anvil &> /dev/null; then
        if pnpm check:foundry | grep -q "Foundry version matches the expected version."; then
            echo "✅ Foundry is already installed with the expected version"
            return
        fi
    fi

    # Install Foundry
    echo "Installing/updating Foundry..."
    if curl -L https://foundry.paradigm.xyz | bash && bash ./install-foundry.sh; then
        echo "✅ Foundry has been installed successfully!"
        forge --version
        cast --version 
        anvil --version
    else
        echo "❌ Foundry installation failed"
        exit 1
    fi
}


install_terraform
install_aws_cli
install_helm
install_kubectl
install_nodejs
install_pnpm
install_cargo
install_foundry