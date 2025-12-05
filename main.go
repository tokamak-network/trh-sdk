package main

import (
	"log"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

const (
	expectedVersion = "1.23.8"
)

func main() {
	currentVersion, err := utils.GetGoVersion()
	if err != nil {
		log.Fatalf("Failed to get Go version, err: %s", err.Error())
	}
	if currentVersion != expectedVersion {
		log.Fatalf("The Go version does not match the expected version: %s. Current version: %s. Please switch to the correct Go version using `gvm use %s`.", expectedVersion, currentVersion, expectedVersion)
	}

	Run()

}
