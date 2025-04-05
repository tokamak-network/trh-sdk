package utils

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os/exec"
	"regexp"
	"sync"
	"time"
)

const (
	base             = 111551119876
	retriedThreshold = 5
)

var chainListURL = "https://chainid.network/chains.json"

var httpClient = &http.Client{Timeout: 10 * time.Second}

// ChainInfo represents chain information returned by the chainlist API.
type ChainInfo struct {
	Name    string `json:"name"`
	ChainID int64  `json:"chainId"`
}

// Cache for chain list data (with mutex for concurrent safety)
var (
	chainListCache     []ChainInfo
	chainListCacheTime time.Time
	cacheMutex         sync.Mutex
	cacheTTL           = 5 * time.Minute // TTL
)

// getChainList fetches chain list data, using cache if valid.
func getChainList() ([]ChainInfo, error) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// If the cache is valid, return the cached data.
	if time.Since(chainListCacheTime) < cacheTTL && chainListCache != nil {
		return chainListCache, nil
	}

	resp, err := httpClient.Get(chainListURL)
	if err != nil {
		return nil, fmt.Errorf("chainlist API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chainlist API returns unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read chainlist API response: %w", err)
	}

	var chains []ChainInfo
	if err = json.Unmarshal(body, &chains); err != nil {
		return nil, fmt.Errorf("chainlist JSON parsing failed: %w", err)
	}

	// Save to cache
	chainListCache = chains
	chainListCacheTime = time.Now()

	return chains, nil
}

// CheckChainIDUsage checks if the given chainID is already in use.
// Returns true if in use, false otherwise.
func CheckChainIDUsage(chainID int64) (bool, error) {
	chains, err := getChainList()
	if err != nil {
		return false, err
	}

	for _, chain := range chains {
		if chain.ChainID == chainID {
			return true, nil
		}
	}
	return false, nil
}

// GenerateL2ChainId generates a unique L2 Chain ID and returns it.
func GenerateL2ChainId() (uint64, error) {
	retried := 0
	for retried < retriedThreshold {
		// Generate a 5-digit random number
		n, err := rand.Int(rand.Reader, big.NewInt(90000))
		if err != nil {
			// If failed, regenerate again
			fmt.Printf("Error initializing random number: %s \n", err.Error())
			retried++
			time.Sleep(100 * time.Millisecond)
			continue
		}
		randomFive := uint64(n.Int64()) + 10000
		candidate := base + randomFive

		// Check if candidate is already in use
		inUse, err := CheckChainIDUsage(int64(candidate))
		if err != nil {
			panic(fmt.Sprintf("failed to check chain id usage: %v", err))
		}
		if !inUse {
			return candidate, nil
		}
		retried++
		time.Sleep(100 * time.Millisecond)
	}

	return 0, fmt.Errorf("failed to generate chain id")
}

func GetGoVersion() (string, error) {
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error executing command:", err)
		return "", err
	}

	versionRegex := regexp.MustCompile(`go(\d+\.\d+\.\d+)`)
	matches := versionRegex.FindStringSubmatch(string(output))

	if len(matches) < 2 {
		fmt.Println("Could not determine Go version.")
		return "", fmt.Errorf("could not determine Go version")
	}

	currentVersion := matches[1]
	return currentVersion, nil
}
