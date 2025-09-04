FROM golang:1.22.6 


WORKDIR /app

# Install base system packages in a single layer
RUN apt-get update && apt-get upgrade -y && \
  apt-get install -y \
  wget \
  gnupg \
  software-properties-common \
  curl \
  unzip \
  jq \
  bc \
  lsb-release \
  ca-certificates \
  sudo \
  build-essential \
  git && \
  # Install Node.js v20.16.0
  curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && \
  apt-get install -y nodejs && \
  npm install -g n && \
  n 20.16.0 && \
  # Clean up in the same layer to reduce image size
  apt-get clean && \
  rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Copy scripts
COPY scripts /app/scripts

# Run setup script
RUN chmod +x /app/scripts/setup.sh && \
  bash -x /app/scripts/setup.sh 2>&1 | tee /var/log/setup.log

SHELL ["/bin/bash", "-c"]

RUN echo 'source ~/.bashrc' >> ~/.bash_profile

CMD ["/bin/bash"]
