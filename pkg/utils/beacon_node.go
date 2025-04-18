package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type BeaconGenesisResponse struct {
	Data struct {
		GenesisTime           string `json:"genesis_time"`
		GenesisValidatorsRoot string `json:"genesis_validators_root"`
		GenesisForkVersion    string `json:"genesis_fork_version"`
	} `json:"data"`
}

// IsValidBeaconURL checks if the L1 beacon URL returns the expected beacon genesis JSON structure
func IsValidBeaconURL(baseURL string) bool {
	if !isValidURL(baseURL) {
		return false
	}
	beaconURL := strings.TrimSuffix(baseURL, "/") + "/eth/v1/beacon/genesis"
	resp, err := http.Get(beaconURL)
	if err != nil {
		fmt.Printf("Error making request to beacon URL: %s\n", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: Unexpected HTTP status %d from beacon URL\n", resp.StatusCode)
		return false
	}

	var beaconResponse BeaconGenesisResponse
	if err := json.NewDecoder(resp.Body).Decode(&beaconResponse); err != nil {
		fmt.Println("Error parsing JSON response from beacon URL:", err)
		return false
	}

	// Ensure expected fields are present
	if beaconResponse.Data.GenesisTime == "" || beaconResponse.Data.GenesisValidatorsRoot == "" || beaconResponse.Data.GenesisForkVersion == "" {
		fmt.Println("Error: Missing required fields in beacon response.")
		return false
	}

	return true
}
