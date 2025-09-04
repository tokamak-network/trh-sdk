FROM ubuntu:latest

WORKDIR /app

# Copy scripts
COPY scripts /app/scripts

# Install base system packages
RUN apt-get update && apt-get upgrade -y && \
  apt-get install -y \
  wget \
  gnupg \
  software-properties-common \
  curl \
  unzip && \
  # Install Node.js v20.16.0
  curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
  apt-get install -y nodejs && \
  npm install -g n && \
  n 20.16.0

# Run setup script
RUN chmod +x /app/scripts/setup.sh && \
  DEBIAN_FRONTEND=noninteractive bash -x /app/scripts/setup.sh 2>&1 | tee /var/log/setup.log


# Clean up
RUN apt-get clean && \
  rm -rf /var/lib/apt/lists/*

# Use bash shell and source profile
SHELL ["/bin/bash", "-c"]
RUN source ~/.bashrc

CMD ["/bin/bash"]
