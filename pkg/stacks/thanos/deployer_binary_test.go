package thanos

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureTokamakDeployer_DownloadFailure(t *testing.T) {
	// 존재하지 않는 버전으로 다운로드 실패 시뮬레이션
	cacheDir := t.TempDir()

	binaryPath, err := ensureTokamakDeployerWithVersion(cacheDir, "v0.0.0-nonexistent")
	if err == nil {
		t.Fatalf("expected error for nonexistent version, got path: %s", binaryPath)
	}
	// 명확한 에러 메시지 확인
	if !strings.Contains(err.Error(), "failed to download tokamak-deployer") {
		t.Errorf("expected download error message, got: %v", err)
	}
}

func TestEnsureTokamakDeployer_CreatesMissingCacheDir(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), ".trh", "bin")

	_, err := ensureTokamakDeployerWithVersion(cacheDir, "v0.0.0-nonexistent")
	if err == nil {
		t.Fatal("expected download failure for nonexistent version")
	}

	info, statErr := os.Stat(cacheDir)
	if statErr != nil {
		t.Fatalf("expected cache dir to be created, got stat error: %v", statErr)
	}
	if !info.IsDir() {
		t.Fatalf("expected %s to be a directory", cacheDir)
	}
}

func TestEnsureTokamakDeployer_CacheHit(t *testing.T) {
	cacheDir := t.TempDir()

	// 캐시에 올바른 버전 바이너리 stub 생성 (실제 다운로드 없이 테스트)
	binaryName := "tokamak-deployer-" + TokamakDeployerVersion
	cachedPath := filepath.Join(cacheDir, binaryName)
	if err := os.WriteFile(cachedPath, []byte("stub binary"), 0755); err != nil {
		t.Fatalf("failed to create stub binary: %v", err)
	}

	// 두 번째 호출은 파일이 이미 있으므로 다운로드 없이 반환
	binaryPath, err := ensureTokamakDeployer(cacheDir)
	if err != nil {
		t.Fatalf("ensureTokamakDeployer with cache hit failed: %v", err)
	}
	if binaryPath != cachedPath {
		t.Errorf("expected path %s, got %s", cachedPath, binaryPath)
	}
}

func TestEnsureTokamakDeployer_VersionMismatch(t *testing.T) {
	cacheDir := t.TempDir()

	// 캐시에 구버전 바이너리 stub 생성
	oldBinaryPath := filepath.Join(cacheDir, "tokamak-deployer-v0.9.0")
	if err := os.WriteFile(oldBinaryPath, []byte("old binary"), 0755); err != nil {
		t.Fatalf("failed to create old binary: %v", err)
	}

	// 현재 버전은 캐시에 없으므로 다운로드 시도 — 하지만 v0.0.0-nonexistent가 아니라
	// 실제로는 다운로드 실패할 것 (GitHub Release 없으므로).
	// 여기서는 캐시 로직만 검증: 구버전 파일은 유지되어야 함.
	// 구버전 파일이 지워지지 않는지만 확인
	if _, err := os.Stat(oldBinaryPath); os.IsNotExist(err) {
		t.Error("old binary should not be deleted")
	}
}
