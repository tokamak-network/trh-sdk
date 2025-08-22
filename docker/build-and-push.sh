#!/bin/bash

# Docker Hub repository
REPO="tokamaknetwork/trh-sdk"

# Tags
AMD64_TAG="itest-reg-amd64"
ARM64_TAG="itest-reg-arm64"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    print_error "Docker is not running. Please start Docker and try again."
    exit 1
fi

# Check if user is logged in to Docker Hub
if ! docker info | grep -q "Username"; then
    print_warning "You are not logged in to Docker Hub. Please run 'docker login' first."
    read -p "Do you want to continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Function to build and push image
build_and_push() {
    local dockerfile=$1
    local tag=$2
    local platform=$3
    
    print_status "Building $platform image from $dockerfile..."
    
    # Build the image
    if docker build --platform $platform -f $dockerfile -t $REPO:$tag .; then
        print_status "Successfully built $REPO:$tag"
        
        # Push the image
        print_status "Pushing $REPO:$tag to Docker Hub..."
        if docker push $REPO:$tag; then
            print_status "Successfully pushed $REPO:$tag to Docker Hub"
        else
            print_error "Failed to push $REPO:$tag to Docker Hub"
            return 1
        fi
    else
        print_error "Failed to build $REPO:$tag"
        return 1
    fi
}

# Main execution
print_status "Starting Docker image build and push process..."

# Build and push AMD64 image
if build_and_push "docker/Dockerfile.amd" $AMD64_TAG "linux/amd64"; then
    print_status "AMD64 image build and push completed successfully"
else
    print_error "AMD64 image build and push failed"
    exit 1
fi

# Build and push ARM64 image
if build_and_push "docker/Dockerfile.arm" $ARM64_TAG "linux/arm64"; then
    print_status "ARM64 image build and push completed successfully"
else
    print_error "ARM64 image build and push failed"
    exit 1
fi

print_status "All images have been built and pushed successfully!"
print_status "Images available at: https://hub.docker.com/r/$REPO/tags"
print_status "Tags: $AMD64_TAG (linux/amd64), $ARM64_TAG (linux/arm64)"
