package thanos

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	npmRegistryURL    = "https://registry.npmjs.org"
	npmPackageName    = "@tokamak-network/thanos-contracts"
	npmArtifactTag    = "dev"
	downloadTimeout   = 120 * time.Second
	maxTarballSize    = 500 * 1024 * 1024 // 500MB safety limit
	maxFileSize       = 100 * 1024 * 1024 // 100MB per-file limit
	forgeArtifactsDir = "forge-artifacts"
	forgeCacheFile    = "cache/solidity-files-cache.json"
)

// npmPackageVersion represents the relevant fields from npm registry version response.
type npmPackageVersion struct {
	Dist struct {
		Tarball string `json:"tarball"`
	} `json:"dist"`
}

// downloadPrebuiltArtifacts downloads pre-built forge artifacts from the npm registry
// and extracts them into the contracts directory. Returns nil on success.
// On any failure, the caller should fall back to building from source.
func downloadPrebuiltArtifacts(ctx context.Context, logger interface{ Info(args ...interface{}) }, contractsDir string) error {
	// Check if artifacts already exist (with cache file to confirm completeness)
	destArtifacts := filepath.Join(contractsDir, forgeArtifactsDir)
	cacheFilePath := filepath.Join(contractsDir, forgeCacheFile)
	if info, err := os.Stat(destArtifacts); err == nil && info.IsDir() {
		entries, _ := os.ReadDir(destArtifacts)
		if len(entries) > 0 {
			if _, cacheErr := os.Stat(cacheFilePath); cacheErr == nil {
				logger.Info("Forge artifacts already exist, skipping download")
				return nil
			}
		}
	}

	logger.Info("Downloading pre-built contract artifacts from npm...")

	// Resolve tarball URL from npm registry
	tarballURL, err := resolveNpmTarballURL(ctx, npmPackageName, npmArtifactTag)
	if err != nil {
		return fmt.Errorf("failed to resolve npm tarball URL: %w", err)
	}

	// Download and extract
	if err := downloadAndExtractArtifacts(ctx, tarballURL, contractsDir); err != nil {
		// Clean up partial extraction
		os.RemoveAll(destArtifacts)
		os.Remove(cacheFilePath)
		// Clean up empty cache directory
		cacheDir := filepath.Dir(cacheFilePath)
		if entries, _ := os.ReadDir(cacheDir); len(entries) == 0 {
			os.Remove(cacheDir)
		}
		return fmt.Errorf("failed to download/extract artifacts: %w", err)
	}

	logger.Info("Pre-built artifacts downloaded successfully")
	return nil
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

// downloadAndExtractArtifacts downloads the npm tarball and extracts forge-artifacts/
// and cache/solidity-files-cache.json into the target directory.
func downloadAndExtractArtifacts(ctx context.Context, tarballURL, contractsDir string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tarballURL, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: downloadTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("tarball download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body) // drain body for connection reuse
		return fmt.Errorf("tarball download returned status %d", resp.StatusCode)
	}

	// npm tarballs are gzipped tar archives with a "package/" prefix
	limitReader := io.LimitReader(resp.Body, maxTarballSize)
	gzReader, err := gzip.NewReader(limitReader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read error: %w", err)
		}

		// npm tarballs have "package/" prefix - strip it
		name := strings.TrimPrefix(header.Name, "package/")

		// Only extract forge-artifacts/ and cache/solidity-files-cache.json
		if !strings.HasPrefix(name, forgeArtifactsDir+"/") && name != forgeCacheFile {
			continue
		}

		// Skip test artifact directories
		if strings.Contains(name, ".t.sol/") {
			continue
		}

		// Reject unreasonably large individual files
		if header.Size > maxFileSize {
			return fmt.Errorf("file %s exceeds size limit (%d > %d bytes)", name, header.Size, maxFileSize)
		}

		targetPath := filepath.Join(contractsDir, name)

		// Prevent path traversal
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(contractsDir)) {
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", targetPath, err)
			}
			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}
			if _, err := io.Copy(outFile, io.LimitReader(tarReader, maxFileSize)); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}
			outFile.Close()
		}
	}

	// Verify extraction produced artifacts
	if _, err := os.Stat(filepath.Join(contractsDir, forgeArtifactsDir)); os.IsNotExist(err) {
		return fmt.Errorf("extraction completed but forge-artifacts directory not found")
	}

	return nil
}

// invalidateCacheEntry removes a specific file entry from the forge solidity-files-cache.json.
// This forces forge to recompile only the specified file while keeping other artifacts intact.
// Uses map[string]json.RawMessage to preserve all top-level fields (paths, builds, profiles, etc.)
// that the npm-downloaded cache contains, preventing forge from triggering a full recompilation.
func invalidateCacheEntry(contractsDir, solFilePath string) error {
	cachePath := filepath.Join(contractsDir, "cache", "solidity-files-cache.json")

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No cache file, nothing to invalidate
		}
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	// Use generic map to preserve ALL top-level fields (paths, builds, profiles, etc.)
	var cacheData map[string]json.RawMessage
	if err := json.Unmarshal(data, &cacheData); err != nil {
		return fmt.Errorf("failed to parse cache structure: %w", err)
	}

	filesRaw, ok := cacheData["files"]
	if !ok {
		return nil // No files field, nothing to invalidate
	}

	var files map[string]json.RawMessage
	if err := json.Unmarshal(filesRaw, &files); err != nil {
		return fmt.Errorf("failed to parse files field: %w", err)
	}

	delete(files, solFilePath)

	updatedFiles, err := json.Marshal(files)
	if err != nil {
		return fmt.Errorf("failed to marshal files: %w", err)
	}
	cacheData["files"] = updatedFiles

	updatedData, err := json.Marshal(cacheData)
	if err != nil {
		return fmt.Errorf("failed to marshal updated cache: %w", err)
	}

	if err := os.WriteFile(cachePath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated cache: %w", err)
	}

	return nil
}
