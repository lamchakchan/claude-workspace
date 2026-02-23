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
	"github.com/lamchakchan/claude-workspace/internal/tools"
)

// ErrMutuallyExclusive is returned when --self-only and --cli-only are both set.
var ErrMutuallyExclusive = fmt.Errorf("--self-only and --cli-only are mutually exclusive")

// ErrUpdateAvailable is returned when --check detects an available update (exit 1).
var ErrUpdateAvailable = fmt.Errorf("update available")

// upgradeFlags holds parsed flags for the upgrade command.
type upgradeFlags struct {
	checkOnly bool
	autoYes   bool
	selfOnly  bool
	cliOnly   bool
}

// parseFlags parses upgrade command arguments into an upgradeFlags struct.
func parseFlags(args []string) (upgradeFlags, error) {
	var f upgradeFlags
	for _, arg := range args {
		switch arg {
		case "--check":
			f.checkOnly = true
		case "--yes", "-y":
			f.autoYes = true
		case "--self-only":
			f.selfOnly = true
		case "--cli-only":
			f.cliOnly = true
		}
	}
	if f.selfOnly && f.cliOnly {
		return f, ErrMutuallyExclusive
	}
	return f, nil
}

// stepCount returns the total number of upgrade steps based on flags.
func stepCount(f upgradeFlags) int {
	if f.selfOnly {
		return 5
	}
	if f.cliOnly {
		return 1
	}
	return 6
}

// Run is the entry point for the upgrade command.
func Run(version string, args []string) error {
	f, err := parseFlags(args)
	if err != nil {
		return err
	}

	checkOnly := f.checkOnly
	autoYes := f.autoYes
	selfOnly := f.selfOnly
	cliOnly := f.cliOnly

	totalSteps := stepCount(f)
	step := 0
	nextStep := func() int {
		step++
		return step
	}

	if !cliOnly {
		platform.PrintBanner(os.Stdout, "Upgrading claude-workspace")

		// Step 1: Check for updates
		platform.PrintStep(os.Stdout, nextStep(), totalSteps, "Checking for updates...")
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
			platform.PrintWarningLine(os.Stdout, "You are running a dev build.")
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
			return ErrUpdateAvailable
		}

		if checkOnly && !selfOnly {
			// In check mode with CLI step, show self status then check CLI
			fmt.Println()
			fmt.Println("  Update available for claude-workspace.")
			if err := upgradeCLI(nextStep, totalSteps, autoYes, checkOnly); err != nil {
				return err
			}
			return ErrUpdateAvailable
		}

		// Confirm
		if !autoYes {
			fmt.Print("\n"); platform.PrintPrompt(os.Stdout, "  Proceed? [Y/n] ")
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "" && answer != "y" && answer != "yes" {
				fmt.Println("  Upgrade cancelled.")
				return nil
			}
		}

		// Step 2: Download
		platform.PrintStep(os.Stdout, nextStep(), totalSteps, fmt.Sprintf("Downloading claude-workspace %s...", latestVersion))

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
		platform.PrintStep(os.Stdout, nextStep(), totalSteps, "Replacing binary...")
		oldVersion := version
		if err := ReplaceBinary(binaryPath); err != nil {
			return fmt.Errorf("replacing binary: %w", err)
		}
		currentExec, _ := os.Executable()
		installPath, _ := filepath.EvalSymlinks(currentExec)
		fmt.Printf("  %s updated (%s → %s)\n", installPath, oldVersion, latestVersion)

		// Step 4: Refresh shared assets
		platform.PrintStep(os.Stdout, nextStep(), totalSteps, "Refreshing shared assets...")
		if _, err := platform.ExtractForSymlink(); err != nil {
			platform.PrintWarningLine(os.Stdout, fmt.Sprintf("could not refresh shared assets: %v", err))
		} else {
			fmt.Println("  ~/.claude-workspace/assets/ updated")
			fmt.Println("  Symlinked projects will pick up changes automatically.")
		}

		// Step 5: Merge global settings
		platform.PrintStep(os.Stdout, nextStep(), totalSteps, "Merging global settings...")
		if err := mergeGlobalSettings(); err != nil {
			platform.PrintWarningLine(os.Stdout, fmt.Sprintf("could not merge settings: %v", err))
		}

		if selfOnly {
			platform.PrintBanner(os.Stdout, "Upgrade Complete")
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

	platform.PrintBanner(os.Stdout, "Upgrade Complete")
	if !cliOnly {
		fmt.Println("\n  Tip: For projects using copied (non-symlinked) assets,")
		fmt.Println("       run 'claude-workspace attach --force' to refresh.")
	}
	fmt.Println()

	return nil
}

// upgradeCLI detects the current Claude Code CLI and runs the official installer to upgrade it.
func upgradeCLI(nextStep func() int, totalSteps int, autoYes, checkOnly bool) error {
	platform.PrintStep(os.Stdout, nextStep(), totalSteps, "Upgrading Claude Code CLI...")

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
		fmt.Print("\n"); platform.PrintPrompt(os.Stdout, fmt.Sprintf("  %s Claude Code CLI? [Y/n] ", action))
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "" && answer != "y" && answer != "yes" {
			fmt.Println("  Skipped Claude Code CLI upgrade.")
			return nil
		}
	}

	// Check for npm-installed Claude before running installer
	npmInfo := setup.DetectNpmClaude()
	if npmInfo.Detected {
		fmt.Printf("  Detected Claude Code installed via npm (source: %s).\n", npmInfo.Source)
		fmt.Println("  Removing npm version before upgrading...")
		if err := setup.UninstallNpmClaude(npmInfo); err != nil {
			platform.PrintWarningLine(os.Stdout, fmt.Sprintf("could not remove npm Claude: %v", err))
			fmt.Println("  You may need to run: npm uninstall -g @anthropic-ai/claude-code")
		} else {
			fmt.Println("  npm Claude Code removed successfully.")
		}
	}

	// Run official installer
	fmt.Println("  Running official installer...")
	if err := platform.Run("bash", "-c", tools.ClaudeInstallCmd); err != nil {
		platform.PrintWarningLine(os.Stdout, fmt.Sprintf("Claude Code CLI upgrade failed: %v", err))
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
