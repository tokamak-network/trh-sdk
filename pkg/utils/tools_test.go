package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// clearChainListCache resets the cached chain list data.
func clearChainListCache() {
	chainListCache = nil
	chainListCacheTime = time.Time{}
}

// Test function of CheckChainIDUsage()
func TestCheckChainIDUsage(t *testing.T) {
	// Example JSON in test server
	chains := []ChainInfo{
		{Name: "TestChain1", ChainID: 11155111}, // in use
	}
	jsonData, err := json.Marshal(chains)
	if err != nil {
		t.Fatalf("Failed to marshal test JSON: %v", err)
	}

	// Create mock server using httptest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(jsonData)
	}))
	defer ts.Close()

	// Override chainListURL with the URL of the ts (test server)
	originalURL := chainListURL
	chainListURL = ts.URL
	clearChainListCache()
	defer func() {
		chainListURL = originalURL
		clearChainListCache()
	}()

	// It must be return True
	inUse, err := CheckChainIDUsage(11155111)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if !inUse {
		t.Errorf("Chain ID 11155111 must be in use.")
	}

	// Chain IDs that are not in the chain list should return false.
	inUse, err = CheckChainIDUsage(111551119876)
	if err != nil {
		t.Errorf("Error: %v", err)
	}
	if inUse {
		t.Errorf("Chain ID 111551119876 must not be in use.")
	}
}

// Test function of GenerateL2ChainId()
func TestGenerateL2ChainId(t *testing.T) {
	// For testing: empty chain list JSON (i.e. assuming all candidates are unused)
	var emptyChains []ChainInfo
	jsonData, err := json.Marshal(emptyChains)
	if err != nil {
		t.Fatalf("JSON marshaling failed: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(jsonData)
	}))
	defer ts.Close()

	originalURL := chainListURL
	chainListURL = ts.URL
	clearChainListCache()
	defer func() {
		chainListURL = originalURL
		clearChainListCache()
	}()

	// call GenerateL2ChainId()
	candidate, err := GenerateL2ChainId()
	require.NoError(t, err)

	t.Logf("Generated L2 Chain ID: %d\n", candidate)

	// Check the range of candidate values based on the base value and the 5-digit random number range
	minCandidate := base + 10000
	maxCandidate := base + 99999

	if candidate < uint64(minCandidate) || candidate > uint64(maxCandidate) {
		t.Errorf("Generated candidate %d is not in the expected range [%d, %d].", candidate, minCandidate, maxCandidate)
	}

	// Check that multiple calls produce different values each time.
	candidate2, err := GenerateL2ChainId()
	require.NoError(t, err)

	t.Logf("Regenerated L2 Chain ID: %d\n", candidate2)
	if candidate == candidate2 {
		t.Errorf("The same candidate was generated in consecutive calls: %d", candidate)
	}

	// Check if the results come out within a certain amount of time when making repeated calls.
	start := time.Now()
	_, err = GenerateL2ChainId()
	require.NoError(t, err)
	elapsed := time.Since(start)
	if elapsed > 5*time.Second {
		t.Errorf("GenerateL2ChainId() function took too long to execute: %v", elapsed)
	}
}
