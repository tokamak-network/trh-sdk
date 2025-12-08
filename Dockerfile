FROM golang:1.24.11 


WORKDIR /app

# Install base system packages in a single layer
RUN apt-get update && \
  apt-get install -y --no-install-recommends \
  wget \
  gnupg \
  curl \
  unzip \
  jq \
  bc \
  lsb-release \
  ca-certificates \
  sudo \
  build-essential \
  git && \
  # Install Node.js v20.16.0 using NodeSource repository
  # Import NodeSource GPG key
  mkdir -p /etc/apt/keyrings && \
  curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg && \
  # Add NodeSource repository
  echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_20.x nodistro main" > /etc/apt/sources.list.d/nodesource.list && \
  apt-get update && \
  apt-get install -y --no-install-recommends nodejs && \
  npm install -g n && \
  n 20.16.0 && \
  # Clean up in the same layer to reduce image size
  apt-get clean && \
  rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

COPY scripts /app/scripts
COPY setup.sh /app/scripts/

# Run setup script
RUN chmod +x /app/scripts/setup.sh && \
  bash -x /app/scripts/setup.sh 2>&1 | tee /var/log/setup.log

SHELL ["/bin/bash", "-c"]

RUN echo 'source ~/.bashrc' >> ~/.bash_profile

CMD ["/bin/bash"]
