package upgrade

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

const releasesURL = "https://api.github.com/repos/lamchakchan/claude-workspace/releases/latest"

// Release represents a GitHub release.
type Release struct {
	TagName     string         `json:"tag_name"`
	Body        string         `json:"body"`
	PublishedAt string         `json:"published_at"`
	Assets      []ReleaseAsset `json:"assets"`
}

// ReleaseAsset represents a downloadable file in a GitHub release.
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// FetchLatest fetches the latest release metadata from GitHub.
func FetchLatest() (*Release, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", releasesURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 || resp.StatusCode == 429 {
		return nil, fmt.Errorf("GitHub API rate limited. Try again later")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("parsing release response: %w", err)
	}
	return &release, nil
}

// FindAsset finds the release asset matching the current OS and architecture.
func FindAsset(release *Release) (*ReleaseAsset, error) {
	osName := runtime.GOOS
	archName := runtime.GOARCH

	// GoReleaser convention: claude-workspace_VERSION_OS_ARCH.tar.gz
	version := strings.TrimPrefix(release.TagName, "v")
	expected := fmt.Sprintf("claude-workspace_%s_%s_%s.tar.gz", version, osName, archName)

	for i := range release.Assets {
		if release.Assets[i].Name == expected {
			return &release.Assets[i], nil
		}
	}

	return nil, fmt.Errorf("no release asset found for %s/%s (expected %s)", osName, archName, expected)
}

// DownloadAsset downloads a release asset to the given destination path.
func DownloadAsset(asset ReleaseAsset, dest string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(asset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", asset.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	sizeMB := float64(asset.Size) / 1024 / 1024
	fmt.Printf("  %s [%.1f MB] ", asset.Name, sizeMB)

	// Copy with progress indicator
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		os.Remove(dest)
		return fmt.Errorf("writing download: %w", err)
	}

	fmt.Println("done.")

	if asset.Size > 0 && written != asset.Size {
		os.Remove(dest)
		return fmt.Errorf("download incomplete: got %d bytes, expected %d", written, asset.Size)
	}

	return nil
}

// VerifyChecksum verifies the downloaded file against checksums.txt from the release.
func VerifyChecksum(release *Release, filePath, assetName string) error {
	// Find checksums.txt asset
	var checksumAsset *ReleaseAsset
	for i := range release.Assets {
		if release.Assets[i].Name == "checksums.txt" {
			checksumAsset = &release.Assets[i]
			break
		}
	}
	if checksumAsset == nil {
		fmt.Println("  No checksums.txt found in release, skipping verification.")
		return nil
	}

	// Download checksums.txt
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(checksumAsset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("fetching checksums.txt: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading checksums.txt: %w", err)
	}

	// Parse checksums.txt: each line is "HASH  FILENAME"
	var expectedHash string
	for _, line := range strings.Split(string(body), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == assetName {
			expectedHash = parts[0]
			break
		}
	}
	if expectedHash == "" {
		return fmt.Errorf("no checksum found for %s in checksums.txt", assetName)
	}

	// Compute actual hash
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("computing checksum: %w", err)
	}
	actualHash := hex.EncodeToString(h.Sum(nil))

	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	fmt.Println("  Checksum verified.")
	return nil
}
