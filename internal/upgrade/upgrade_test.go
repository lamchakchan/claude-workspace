package upgrade

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

func TestFetchLatestWithMockServer(t *testing.T) {
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
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("missing Accept header, got %q", r.Header.Get("Accept"))
		}
		_ = json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	origURL := ReleasesURL
	ReleasesURL = server.URL
	defer func() { ReleasesURL = origURL }()

	got, err := FetchLatest()
	if err != nil {
		t.Fatalf("FetchLatest() error = %v", err)
	}
	if got.TagName != "v1.5.0" {
		t.Errorf("TagName = %s, want v1.5.0", got.TagName)
	}
	if len(got.Assets) != 2 {
		t.Errorf("Assets count = %d, want 2", len(got.Assets))
	}
	if got.Body != release.Body {
		t.Errorf("Body = %q, want %q", got.Body, release.Body)
	}
}

func TestFetchLatestRateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(403)
	}))
	defer server.Close()

	origURL := ReleasesURL
	ReleasesURL = server.URL
	defer func() { ReleasesURL = origURL }()

	_, err := FetchLatest()
	if err == nil {
		t.Fatal("expected error for rate-limited response")
	}
	if !strings.Contains(err.Error(), "rate limited") {
		t.Errorf("expected rate limit error, got: %v", err)
	}
}

func TestFetchLatestServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()

	origURL := ReleasesURL
	ReleasesURL = server.URL
	defer func() { ReleasesURL = origURL }()

	_, err := FetchLatest()
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("expected status 500 error, got: %v", err)
	}
}

func TestDownloadAsset(t *testing.T) {
	content := "binary-content-here"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(content))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "download.tar.gz")

	asset := ReleaseAsset{
		Name:               "test.tar.gz",
		BrowserDownloadURL: server.URL,
		Size:               int64(len(content)),
	}

	if err := DownloadAsset(asset, dest); err != nil {
		t.Fatalf("DownloadAsset() error = %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading downloaded file: %v", err)
	}
	if string(got) != content {
		t.Errorf("downloaded content = %q, want %q", string(got), content)
	}
}

func TestDownloadAssetServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "download.tar.gz")

	asset := ReleaseAsset{
		Name:               "test.tar.gz",
		BrowserDownloadURL: server.URL,
		Size:               100,
	}

	err := DownloadAsset(asset, dest)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "status 404") {
		t.Errorf("expected status 404 error, got: %v", err)
	}
}

func TestVerifyChecksum(t *testing.T) {
	// Create a test file and compute its checksum
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.tar.gz")
	fileContent := []byte("test-archive-content")
	_ = os.WriteFile(filePath, fileContent, 0644)

	h := sha256.Sum256(fileContent)
	expectedHash := hex.EncodeToString(h[:])
	checksumContent := fmt.Sprintf("%s  test.tar.gz\n%s  other.tar.gz\n", expectedHash, "0000000000000000000000000000000000000000000000000000000000000000")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(checksumContent))
	}))
	defer server.Close()

	release := &Release{
		Assets: []ReleaseAsset{
			{Name: "test.tar.gz"},
			{Name: "checksums.txt", BrowserDownloadURL: server.URL},
		},
	}

	if err := VerifyChecksum(release, filePath, "test.tar.gz"); err != nil {
		t.Fatalf("VerifyChecksum() error = %v", err)
	}
}

func TestVerifyChecksumMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.tar.gz")
	_ = os.WriteFile(filePath, []byte("actual content"), 0644)

	checksumContent := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  test.tar.gz\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(checksumContent))
	}))
	defer server.Close()

	release := &Release{
		Assets: []ReleaseAsset{
			{Name: "checksums.txt", BrowserDownloadURL: server.URL},
		},
	}

	err := VerifyChecksum(release, filePath, "test.tar.gz")
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("expected checksum mismatch error, got: %v", err)
	}
}

func TestVerifyChecksumNoChecksumFile(t *testing.T) {
	release := &Release{
		Assets: []ReleaseAsset{
			{Name: "test.tar.gz"},
		},
	}

	// Should not error — just prints a skip message
	if err := VerifyChecksum(release, "/whatever", "test.tar.gz"); err != nil {
		t.Fatalf("expected nil error when no checksums.txt, got: %v", err)
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

	// Verify the file is executable
	info, err := os.Stat(binaryPath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("extracted binary should be executable")
	}
}

func TestExtractBinaryNestedPath(t *testing.T) {
	// GoReleaser sometimes nests the binary in a directory
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.tar.gz")

	createTestArchive(t, archivePath, "claude-workspace_1.5.0_linux_amd64/claude-workspace", "binary-content")

	binaryPath, err := extractBinary(archivePath, tmpDir)
	if err != nil {
		t.Fatalf("extractBinary() error = %v", err)
	}

	content, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("reading extracted binary: %v", err)
	}
	if string(content) != "binary-content" {
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

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    upgradeFlags
		wantErr error
	}{
		{
			name: "empty args",
			args: []string{},
			want: upgradeFlags{},
		},
		{
			name: "self-only",
			args: []string{"--self-only"},
			want: upgradeFlags{selfOnly: true},
		},
		{
			name: "cli-only",
			args: []string{"--cli-only"},
			want: upgradeFlags{cliOnly: true},
		},
		{
			name:    "mutually exclusive",
			args:    []string{"--self-only", "--cli-only"},
			wantErr: ErrMutuallyExclusive,
		},
		{
			name: "check and yes",
			args: []string{"--check", "--yes"},
			want: upgradeFlags{checkOnly: true, autoYes: true},
		},
		{
			name: "short yes alias",
			args: []string{"-y"},
			want: upgradeFlags{autoYes: true},
		},
		{
			name: "unknown flags ignored",
			args: []string{"--verbose", "--self-only", "--unknown"},
			want: upgradeFlags{selfOnly: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFlags(tt.args)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("parseFlags(%v) error = %v, want %v", tt.args, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseFlags(%v) unexpected error: %v", tt.args, err)
			}
			if got != tt.want {
				t.Errorf("parseFlags(%v) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}

func TestStepCount(t *testing.T) {
	tests := []struct {
		name  string
		flags upgradeFlags
		want  int
	}{
		{name: "default", flags: upgradeFlags{}, want: 6},
		{name: "self-only", flags: upgradeFlags{selfOnly: true}, want: 5},
		{name: "cli-only", flags: upgradeFlags{cliOnly: true}, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stepCount(tt.flags); got != tt.want {
				t.Errorf("stepCount(%+v) = %d, want %d", tt.flags, got, tt.want)
			}
		})
	}
}

func TestRunMutuallyExclusive(t *testing.T) {
	err := Run("dev", []string{"--self-only", "--cli-only"})
	if err != ErrMutuallyExclusive {
		t.Errorf("Run() error = %v, want %v", err, ErrMutuallyExclusive)
	}
}

func TestIsHomebrewBinary(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/opt/homebrew/bin/claude", true},
		{"/usr/local/Cellar/claude-code/2.1.50/bin/claude", true},
		{"/usr/local/opt/claude-code/bin/claude", true},
		{"/home/user/.local/bin/claude", false},
		{"/usr/bin/claude", false},
	}
	for _, c := range cases {
		if got := isHomebrewBinary(c.path); got != c.want {
			t.Errorf("isHomebrewBinary(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestRunSelfOnlyUpToDate(t *testing.T) {
	release := Release{
		TagName: "v1.0.0",
		Assets:  []ReleaseAsset{},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	origURL := ReleasesURL
	ReleasesURL = server.URL
	defer func() { ReleasesURL = origURL }()

	err := Run("1.0.0", []string{"--self-only", "--check"})
	if err != nil {
		t.Errorf("Run(self-only, check, up-to-date) error = %v, want nil", err)
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
