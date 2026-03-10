BINARY    := trh-sdk
PKG       := github.com/tokamak-network/trh-sdk
LDFLAGS   := -ldflags="-s -w"

# Default target
.PHONY: build
build:
	go build $(LDFLAGS) -o $(BINARY) .

# Build for all supported platforms (linux/amd64, linux/arm64, darwin/arm64)
.PHONY: build-all
build-all:
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-linux-amd64 .
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY)-linux-arm64 .
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY)-darwin-arm64 .

.PHONY: test
test:
	go test -race ./...

.PHONY: lint
lint:
	golangci-lint run --config=.golangci.yml --timeout=10m

.PHONY: clean
clean:
	rm -f $(BINARY) $(BINARY)-linux-amd64 $(BINARY)-linux-arm64 $(BINARY)-darwin-arm64
