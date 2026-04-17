package thanos

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// newHTTPRequest creates a GET request with context.
func newHTTPRequest(ctx context.Context, url string) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
}

// newHTTPClient creates an HTTP client with the given timeout.
func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

// extractFileFromTarball reads a gzipped tar stream and returns the content of the named file.
func extractFileFromTarball(r io.Reader, targetPath string) ([]byte, error) {
	gzReader, err := gzip.NewReader(io.LimitReader(r, maxTarballSize))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar read error: %w", err)
		}

		// Normalize path: npm tarballs may or may not have leading "./"
		name := strings.TrimPrefix(header.Name, "./")

		// Skip entries with path traversal attempts
		if strings.Contains(name, "..") {
			continue
		}

		if name == targetPath {
			if header.Size > maxFileSize {
				return nil, fmt.Errorf("file %s exceeds size limit (%d > %d)", name, header.Size, maxFileSize)
			}
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, io.LimitReader(tarReader, maxFileSize)); err != nil {
				return nil, fmt.Errorf("failed to read %s: %w", name, err)
			}
			return buf.Bytes(), nil
		}
	}

	return nil, fmt.Errorf("file %s not found in tarball", targetPath)
}
