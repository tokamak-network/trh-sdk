package main

import (
	"fmt"
	"log"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

const (
	expectedVersion = "1.22.6"
)

func main() {
	currentVersion, err := utils.GetGoVersion()
	if err != nil {
		log.Fatalf("Failed to get Go version, err: %s", err.Error())
	}
	if currentVersion != expectedVersion {
		log.Fatal(fmt.Sprintf("Go version does not match expected version: %s, current version: %s", expectedVersion, currentVersion))
	}

	Run()
}
