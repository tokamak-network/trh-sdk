package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	npmRegistryURL  = "https://registry.npmjs.org"
	downloadTimeout = 120 * time.Second
	maxTarballSize  = 500 * 1024 * 1024 // 500MB safety limit
	maxFileSize     = 100 * 1024 * 1024 // 100MB per-file limit
)

// npmPackageVersion represents the relevant fields from npm registry version response.
type npmPackageVersion struct {
	Dist struct {
		Tarball string `json:"tarball"`
	} `json:"dist"`
}

// resolveNpmTarballURL queries the npm registry for the tarball URL of a specific package tag.
func resolveNpmTarballURL(ctx context.Context, packageName, tag string) (string, error) {
	// npm registry accepts scoped packages with literal @ and / in the URL path
	registryURL := fmt.Sprintf("%s/%s/%s", npmRegistryURL, packageName, tag)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, registryURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("npm registry request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body) // drain body for connection reuse
		return "", fmt.Errorf("npm registry returned status %d for %s@%s", resp.StatusCode, packageName, tag)
	}

	var version npmPackageVersion
	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		return "", fmt.Errorf("failed to parse npm registry response: %w", err)
	}

	if version.Dist.Tarball == "" {
		return "", fmt.Errorf("no tarball URL found for %s@%s", packageName, tag)
	}

	return version.Dist.Tarball, nil
}
