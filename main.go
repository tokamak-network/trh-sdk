package main

import (
	"fmt"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"log"
)

const (
	expectedVersion = "1.22.6"
)

func main() {
	currentVersion, err := utils.GetGoVersion()
	if err != nil {
		log.Fatalf("Failed to get Go version, err: %s", err.Error())
	}
	if currentVersion == expectedVersion {
		fmt.Println("Go version is correct:", currentVersion)
	} else {
		log.Fatal(fmt.Sprintf("Go version does not match expected version: %s, current version: %s", expectedVersion, currentVersion))
	}

	Run()
}
