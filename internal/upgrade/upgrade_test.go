package upgrade

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFindAsset(t *testing.T) {
	version := "1.5.0"
	osName := runtime.GOOS
	archName := runtime.GOARCH
	expectedName := "claude-workspace_" + version + "_" + osName + "_" + archName + ".tar.gz"

	release := &Release{
		TagName: "v" + version,
		Assets: []ReleaseAsset{
			{Name: "claude-workspace_1.5.0_linux_amd64.tar.gz", BrowserDownloadURL: "https://example.com/linux"},
			{Name: "claude-workspace_1.5.0_darwin_arm64.tar.gz", BrowserDownloadURL: "https://example.com/darwin"},
			{Name: "claude-workspace_1.5.0_darwin_amd64.tar.gz", BrowserDownloadURL: "https://example.com/darwin_amd64"},
			{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums"},
		},
	}

	asset, err := FindAsset(release)
	if err != nil {
		t.Fatalf("FindAsset() error = %v", err)
	}
	if asset.Name != expectedName {
		t.Errorf("FindAsset() got %s, want %s", asset.Name, expectedName)
	}
}

func TestFindAssetNotFound(t *testing.T) {
	release := &Release{
		TagName: "v1.5.0",
		Assets: []ReleaseAsset{
			{Name: "claude-workspace_1.5.0_windows_amd64.zip"},
		},
	}

	_, err := FindAsset(release)
	if err == nil {
		t.Fatal("FindAsset() expected error for missing asset")
	}
	if !strings.Contains(err.Error(), "no release asset found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFetchLatestParsing(t *testing.T) {
	release := Release{
		TagName:     "v1.5.0",
		Body:        "- Added context-manager skill\n- Updated security hook",
		PublishedAt: "2026-02-20T00:00:00Z",
		Assets: []ReleaseAsset{
			{Name: "claude-workspace_1.5.0_darwin_arm64.tar.gz", Size: 10000000},
			{Name: "checksums.txt", Size: 500},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	var got Release
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if got.TagName != "v1.5.0" {
		t.Errorf("TagName = %s, want v1.5.0", got.TagName)
	}
	if len(got.Assets) != 2 {
		t.Errorf("Assets count = %d, want 2", len(got.Assets))
	}
}

func TestExtractBinary(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.tar.gz")

	createTestArchive(t, archivePath, "claude-workspace", "#!/bin/sh\necho test")

	binaryPath, err := extractBinary(archivePath, tmpDir)
	if err != nil {
		t.Fatalf("extractBinary() error = %v", err)
	}

	content, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("reading extracted binary: %v", err)
	}
	if string(content) != "#!/bin/sh\necho test" {
		t.Errorf("unexpected content: %s", string(content))
	}
}

func TestExtractBinaryNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.tar.gz")

	createTestArchive(t, archivePath, "other-binary", "content")

	_, err := extractBinary(archivePath, tmpDir)
	if err == nil {
		t.Fatal("expected error for missing binary in archive")
	}
	if !strings.Contains(err.Error(), "not found in archive") {
		t.Errorf("unexpected error: %v", err)
	}
}

func createTestArchive(t *testing.T, archivePath, fileName, content string) {
	t.Helper()

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	hdr := &tar.Header{
		Name: fileName,
		Mode: 0755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
}
