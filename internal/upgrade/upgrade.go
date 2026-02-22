package upgrade

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
	"github.com/lamchakchan/claude-workspace/internal/setup"
)

// ErrMutuallyExclusive is returned when --self-only and --cli-only are both set.
var ErrMutuallyExclusive = fmt.Errorf("--self-only and --cli-only are mutually exclusive")

// Run is the entry point for the upgrade command.
func Run(version string, args []string) error {
	checkOnly := false
	autoYes := false
	selfOnly := false
	cliOnly := false

	for _, arg := range args {
		switch arg {
		case "--check":
			checkOnly = true
		case "--yes", "-y":
			autoYes = true
		case "--self-only":
			selfOnly = true
		case "--cli-only":
			cliOnly = true
		}
	}

	if selfOnly && cliOnly {
		return ErrMutuallyExclusive
	}

	// Determine step count and labels
	totalSteps := 6
	if selfOnly {
		totalSteps = 5
	} else if cliOnly {
		totalSteps = 1
	}
	step := 0
	nextStep := func() int {
		step++
		return step
	}

	if !cliOnly {
		fmt.Println("\n=== Upgrading claude-workspace ===")

		// Step 1: Check for updates
		fmt.Printf("\n[%d/%d] Checking for updates...\n", nextStep(), totalSteps)
		fmt.Printf("  Current: %s\n", version)

		release, err := FetchLatest()
		if err != nil {
			return fmt.Errorf("checking for updates: %w", err)
		}

		latestVersion := release.TagName
		publishedDate := ""
		if release.PublishedAt != "" {
			// Trim to date portion
			if idx := strings.Index(release.PublishedAt, "T"); idx > 0 {
				publishedDate = release.PublishedAt[:idx]
			} else {
				publishedDate = release.PublishedAt
			}
		}

		fmt.Printf("  Latest:  %s", latestVersion)
		if publishedDate != "" {
			fmt.Printf(" (%s)", publishedDate)
		}
		fmt.Println()

		// Compare versions
		currentNormalized := version
		if !strings.HasPrefix(currentNormalized, "v") {
			currentNormalized = "v" + currentNormalized
		}

		if version == "dev" {
			fmt.Println("\n  Warning: You are running a dev build.")
			fmt.Println("  Upgrading will install the latest stable release.")
		} else if currentNormalized == latestVersion {
			fmt.Println("\n  Already up to date.")
			if !selfOnly {
				// Still run CLI upgrade step
				return upgradeCLI(nextStep, totalSteps, autoYes, checkOnly)
			}
			return nil
		}

		// Print changelog
		if release.Body != "" {
			fmt.Println("\n  Changelog:")
			for _, line := range strings.Split(release.Body, "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					fmt.Printf("    %s\n", line)
				}
			}
		}

		if checkOnly && selfOnly {
			fmt.Println()
			os.Exit(1) // exit 1 = update available (as documented)
		}

		if checkOnly && !selfOnly {
			// In check mode with CLI step, show self status then check CLI
			fmt.Println()
			fmt.Println("  Update available for claude-workspace.")
			return upgradeCLI(nextStep, totalSteps, autoYes, checkOnly)
		}

		// Confirm
		if !autoYes {
			fmt.Print("\n  Proceed? [Y/n] ")
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "" && answer != "y" && answer != "yes" {
				fmt.Println("  Upgrade cancelled.")
				return nil
			}
		}

		// Step 2: Download
		fmt.Printf("\n[%d/%d] Downloading claude-workspace %s...\n", nextStep(), totalSteps, latestVersion)

		asset, err := FindAsset(release)
		if err != nil {
			return err
		}

		tmpDir, err := os.MkdirTemp("", "claude-workspace-upgrade-*")
		if err != nil {
			return fmt.Errorf("creating temp directory: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		archivePath := filepath.Join(tmpDir, asset.Name)
		if err := DownloadAsset(*asset, archivePath); err != nil {
			return err
		}

		if err := VerifyChecksum(release, archivePath, asset.Name); err != nil {
			return err
		}

		// Extract binary from tarball
		binaryPath, err := extractBinary(archivePath, tmpDir)
		if err != nil {
			return fmt.Errorf("extracting binary: %w", err)
		}

		// Step 3: Replace binary
		fmt.Printf("\n[%d/%d] Replacing binary...\n", nextStep(), totalSteps)
		oldVersion := version
		if err := ReplaceBinary(binaryPath); err != nil {
			return fmt.Errorf("replacing binary: %w", err)
		}
		currentExec, _ := os.Executable()
		installPath, _ := filepath.EvalSymlinks(currentExec)
		fmt.Printf("  %s updated (%s → %s)\n", installPath, oldVersion, latestVersion)

		// Step 4: Refresh shared assets
		fmt.Printf("\n[%d/%d] Refreshing shared assets...\n", nextStep(), totalSteps)
		if _, err := platform.ExtractForSymlink(); err != nil {
			fmt.Printf("  Warning: could not refresh shared assets: %v\n", err)
		} else {
			fmt.Println("  ~/.claude-workspace/assets/ updated")
			fmt.Println("  Symlinked projects will pick up changes automatically.")
		}

		// Step 5: Merge global settings
		fmt.Printf("\n[%d/%d] Merging global settings...\n", nextStep(), totalSteps)
		if err := mergeGlobalSettings(); err != nil {
			fmt.Printf("  Warning: could not merge settings: %v\n", err)
		}

		if selfOnly {
			fmt.Println("\n=== Upgrade Complete ===")
			fmt.Println("\n  Tip: For projects using copied (non-symlinked) assets,")
			fmt.Println("       run 'claude-workspace attach --force' to refresh.")
			fmt.Println()
			return nil
		}
	}

	// Final step: Upgrade Claude Code CLI
	if err := upgradeCLI(nextStep, totalSteps, autoYes, checkOnly); err != nil {
		return err
	}

	fmt.Println("\n=== Upgrade Complete ===")
	if !cliOnly {
		fmt.Println("\n  Tip: For projects using copied (non-symlinked) assets,")
		fmt.Println("       run 'claude-workspace attach --force' to refresh.")
	}
	fmt.Println()

	return nil
}

// upgradeCLI detects the current Claude Code CLI and runs the official installer to upgrade it.
func upgradeCLI(nextStep func() int, totalSteps int, autoYes, checkOnly bool) error {
	fmt.Printf("\n[%d/%d] Upgrading Claude Code CLI...\n", nextStep(), totalSteps)

	home, _ := os.UserHomeDir()

	// Detect claude binary (reuse pattern from doctor.go)
	claudeBin := "claude"
	if !platform.Exists(claudeBin) {
		localBin := filepath.Join(home, ".local", "bin", "claude")
		if platform.FileExists(localBin) {
			claudeBin = localBin
		}
	}

	installed := false
	oldVersion := ""
	if ver, err := platform.Output(claudeBin, "--version"); err == nil {
		installed = true
		oldVersion = strings.TrimSpace(ver)
		fmt.Printf("  Current Claude Code CLI: %s\n", oldVersion)
	} else {
		fmt.Println("  Claude Code CLI not found.")
	}

	if checkOnly {
		if installed {
			fmt.Println("  (Cannot check latest version remotely; run without --check to upgrade.)")
		} else {
			fmt.Println("  Claude Code CLI is not installed.")
		}
		return nil
	}

	// Prompt user
	if !autoYes {
		action := "Install"
		if installed {
			action = "Upgrade"
		}
		fmt.Printf("\n  %s Claude Code CLI? [Y/n] ", action)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "" && answer != "y" && answer != "yes" {
			fmt.Println("  Skipped Claude Code CLI upgrade.")
			return nil
		}
	}

	// Run official installer
	fmt.Println("  Running official installer...")
	if err := platform.Run("bash", "-c", "curl -fsSL https://claude.ai/install.sh | bash"); err != nil {
		fmt.Printf("  Warning: Claude Code CLI upgrade failed: %v\n", err)
		fmt.Println("  You can upgrade manually: curl -fsSL https://claude.ai/install.sh | bash")
		return nil // non-fatal
	}

	// Augment PATH for current process so we can detect the new version
	localBin := filepath.Join(home, ".local", "bin")
	if platform.FileExists(filepath.Join(localBin, "claude")) {
		os.Setenv("PATH", localBin+":"+os.Getenv("PATH"))
	}

	// Show version comparison
	if newVer, err := platform.Output(claudeBin, "--version"); err == nil {
		newVersion := strings.TrimSpace(newVer)
		if oldVersion != "" {
			fmt.Printf("  Claude Code CLI: %s → %s\n", oldVersion, newVersion)
		} else {
			fmt.Printf("  Claude Code CLI installed: %s\n", newVersion)
		}
	} else {
		fmt.Println("  Claude Code CLI installed successfully.")
	}

	return nil
}

// mergeGlobalSettings merges platform defaults into ~/.claude/settings.json.
func mergeGlobalSettings() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	defaults := setup.GetDefaultGlobalSettings()

	if !platform.FileExists(settingsPath) {
		fmt.Println("  No global settings found, skipping merge.")
		return nil
	}

	var existing map[string]interface{}
	if err := platform.ReadJSONFile(settingsPath, &existing); err != nil {
		return fmt.Errorf("reading settings: %w", err)
	}

	merged := setup.MergeSettings(existing, defaults)
	if err := platform.WriteJSONFile(settingsPath, merged); err != nil {
		return fmt.Errorf("writing settings: %w", err)
	}

	fmt.Println("  ~/.claude/settings.json: defaults merged")
	return nil
}

// extractBinary extracts the claude-workspace binary from a .tar.gz archive.
func extractBinary(archivePath, destDir string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("opening gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("reading tar: %w", err)
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		baseName := filepath.Base(header.Name)
		if baseName == "claude-workspace" {
			destPath := filepath.Join(destDir, "claude-workspace")
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return "", fmt.Errorf("extracting binary: %w", err)
			}
			out.Close()
			return destPath, nil
		}
	}

	return "", fmt.Errorf("claude-workspace binary not found in archive")
}
