// Package upgrade implements the "upgrade" command for self-updating the
// claude-workspace binary from GitHub releases and upgrading the Claude Code CLI.
package upgrade

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
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

// stepper tracks step progress for multi-step commands.
type stepper struct {
	total   int
	current int
}

func newStepper(f upgradeFlags) *stepper {
	return &stepper{total: stepCount(f)}
}

func (s *stepper) next() int {
	s.current++
	return s.current
}

// Run is the entry point for the upgrade command.
func Run(version string, args []string) error {
	f, err := parseFlags(args)
	if err != nil {
		return err
	}

	s := newStepper(f)

	if !f.cliOnly {
		if err := upgradeSelf(version, f, s); err != nil {
			return err
		}
		if f.selfOnly {
			printUpgradeComplete(false)
			return nil
		}
	}

	if err := upgradeCLI(s, f.autoYes, f.checkOnly); err != nil {
		return err
	}

	printUpgradeComplete(!f.cliOnly)
	return nil
}

// upgradeSelf handles the self-update portion (steps 1-5).
func upgradeSelf(version string, f upgradeFlags, s *stepper) error {
	platform.PrintBanner(os.Stdout, "Upgrading claude-workspace")

	release, upToDate, err := checkForUpdates(version, s)
	if err != nil {
		return err
	}

	if upToDate {
		if !f.selfOnly {
			return upgradeCLI(s, f.autoYes, f.checkOnly)
		}
		return nil
	}

	printChangelog(release)

	if f.checkOnly {
		return handleCheckOnly(f, s)
	}

	if !f.autoYes && !confirmUpgrade() {
		return nil
	}

	return downloadAndInstall(version, release, s)
}

// checkForUpdates fetches the latest release and compares versions.
// Returns the release, whether the current version is up to date, and any error.
func checkForUpdates(version string, s *stepper) (*Release, bool, error) {
	platform.PrintStep(os.Stdout, s.next(), s.total, "Checking for updates...")
	fmt.Printf("  Current: %s\n", version)

	release, err := FetchLatest()
	if err != nil {
		return nil, false, fmt.Errorf("checking for updates: %w", err)
	}

	latestVersion := release.TagName
	publishedDate := extractDate(release.PublishedAt)

	fmt.Printf("  Latest:  %s", latestVersion)
	if publishedDate != "" {
		fmt.Printf(" (%s)", publishedDate)
	}
	fmt.Println()

	currentNormalized := version
	if !strings.HasPrefix(currentNormalized, "v") {
		currentNormalized = "v" + currentNormalized
	}

	if version == "dev" {
		platform.PrintWarningLine(os.Stdout, "You are running a dev build.")
		fmt.Println("  Upgrading will install the latest stable release.")
		return release, false, nil
	}

	if currentNormalized == latestVersion {
		fmt.Println("\n  Already up to date.")
		return release, true, nil
	}

	return release, false, nil
}

// extractDate returns the date portion of an ISO timestamp.
func extractDate(ts string) string {
	if idx := strings.Index(ts, "T"); idx > 0 {
		return ts[:idx]
	}
	return ts
}

// printChangelog prints the release changelog if present.
func printChangelog(release *Release) {
	if release.Body == "" {
		return
	}
	fmt.Println("\n  Changelog:")
	for _, line := range strings.Split(release.Body, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			fmt.Printf("    %s\n", line)
		}
	}
}

// handleCheckOnly handles the --check flag behavior.
func handleCheckOnly(f upgradeFlags, s *stepper) error {
	if f.selfOnly {
		fmt.Println()
		return ErrUpdateAvailable
	}
	fmt.Println()
	fmt.Println("  Update available for claude-workspace.")
	if err := upgradeCLI(s, f.autoYes, f.checkOnly); err != nil {
		return err
	}
	return ErrUpdateAvailable
}

// confirmUpgrade prompts the user and returns true if they accept.
func confirmUpgrade() bool {
	fmt.Print("\n")
	platform.PrintPrompt(os.Stdout, "  Proceed? [Y/n] ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "" && answer != "y" && answer != "yes" {
		fmt.Println("  Upgrade cancelled.")
		return false
	}
	return true
}

// downloadAndInstall downloads, verifies, and installs the new binary (steps 2-5).
func downloadAndInstall(version string, release *Release, s *stepper) error {
	latestVersion := release.TagName

	platform.PrintStep(os.Stdout, s.next(), s.total, fmt.Sprintf("Downloading claude-workspace %s...", latestVersion))

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

	binaryPath, err := extractBinary(archivePath, tmpDir)
	if err != nil {
		return fmt.Errorf("extracting binary: %w", err)
	}

	platform.PrintStep(os.Stdout, s.next(), s.total, "Replacing binary...")
	if err := ReplaceBinary(binaryPath); err != nil {
		return fmt.Errorf("replacing binary: %w", err)
	}
	currentExec, _ := os.Executable()
	installPath, _ := filepath.EvalSymlinks(currentExec)
	fmt.Printf("  %s updated (%s → %s)\n", installPath, version, latestVersion)

	refreshAssets(s)
	mergeSettings(s)

	return nil
}

// refreshAssets updates shared symlinked assets (step 4).
func refreshAssets(s *stepper) {
	platform.PrintStep(os.Stdout, s.next(), s.total, "Refreshing shared assets...")
	if _, err := platform.ExtractForSymlink(); err != nil {
		platform.PrintWarningLine(os.Stdout, fmt.Sprintf("could not refresh shared assets: %v", err))
	} else {
		fmt.Println("  ~/.claude-workspace/assets/ updated")
		fmt.Println("  Symlinked projects will pick up changes automatically.")
	}
}

// mergeSettings merges platform defaults into global settings (step 5).
func mergeSettings(s *stepper) {
	platform.PrintStep(os.Stdout, s.next(), s.total, "Merging global settings...")
	if err := mergeGlobalSettings(); err != nil {
		platform.PrintWarningLine(os.Stdout, fmt.Sprintf("could not merge settings: %v", err))
	}
}

// printUpgradeComplete prints the final upgrade banner.
func printUpgradeComplete(showTip bool) {
	platform.PrintBanner(os.Stdout, "Upgrade Complete")
	if showTip {
		fmt.Println("\n  Tip: For projects using copied (non-symlinked) assets,")
		fmt.Println("       run 'claude-workspace attach --force' to refresh.")
	}
	fmt.Println()
}

// cliInfo holds detected Claude Code CLI state.
type cliInfo struct {
	BinPath    string
	IsHomebrew bool
	Installed  bool
	OldVersion string
}

// detectClaudeBinary locates the Claude Code CLI and determines its install method.
func detectClaudeBinary() cliInfo {
	info := cliInfo{BinPath: "claude"}

	if resolvedPath, err := exec.LookPath("claude"); err == nil {
		info.BinPath = resolvedPath
		info.IsHomebrew = isHomebrewBinary(resolvedPath)
	} else {
		home, _ := os.UserHomeDir()
		localBin := filepath.Join(home, ".local", "bin", "claude")
		if platform.FileExists(localBin) {
			info.BinPath = localBin
		}
	}

	if ver, err := platform.Output(info.BinPath, "--version"); err == nil {
		info.Installed = true
		info.OldVersion = strings.TrimSpace(ver)
	}

	return info
}

// runCLIInstall executes the appropriate CLI install/upgrade command.
func runCLIInstall(info cliInfo) {
	if info.IsHomebrew {
		fmt.Println("  Detected Homebrew installation. Running: brew upgrade claude-code...")
		if err := platform.Run("brew", "upgrade", "claude-code"); err != nil {
			fmt.Println("  Claude Code is already up to date (or brew upgrade failed).")
			fmt.Println("  To upgrade manually: brew upgrade claude-code")
		}
		return
	}

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

	fmt.Println("  Running official installer...")
	if err := platform.Run("bash", "-c", tools.ClaudeInstallCmd); err != nil {
		platform.PrintWarningLine(os.Stdout, fmt.Sprintf("Claude Code CLI upgrade failed: %v", err))
		fmt.Println("  You can upgrade manually: curl -fsSL https://claude.ai/install.sh | bash")
		return
	}

	home, _ := os.UserHomeDir()
	localBin := filepath.Join(home, ".local", "bin")
	if platform.FileExists(filepath.Join(localBin, "claude")) {
		os.Setenv("PATH", localBin+":"+os.Getenv("PATH"))
	}
}

// reportCLIVersion prints a before/after version comparison.
func reportCLIVersion(info cliInfo) {
	newVer, err := platform.Output(info.BinPath, "--version")
	if err != nil {
		fmt.Println("  Claude Code CLI installed successfully.")
		return
	}
	newVersion := strings.TrimSpace(newVer)
	if info.OldVersion != "" {
		fmt.Printf("  Claude Code CLI: %s → %s\n", info.OldVersion, newVersion)
	} else {
		fmt.Printf("  Claude Code CLI installed: %s\n", newVersion)
	}
}

// upgradeCLI detects the current Claude Code CLI and runs the official installer to upgrade it.
func upgradeCLI(s *stepper, autoYes, checkOnly bool) error {
	platform.PrintStep(os.Stdout, s.next(), s.total, "Upgrading Claude Code CLI...")

	info := detectClaudeBinary()

	if info.Installed {
		fmt.Printf("  Current Claude Code CLI: %s\n", info.OldVersion)
	} else {
		fmt.Println("  Claude Code CLI not found.")
	}

	if checkOnly {
		if info.Installed {
			fmt.Println("  (Cannot check latest version remotely; run without --check to upgrade.)")
		} else {
			fmt.Println("  Claude Code CLI is not installed.")
		}
		return nil
	}

	if !autoYes {
		action := "Install"
		if info.Installed {
			action = "Upgrade"
		}
		fmt.Print("\n")
		platform.PrintPrompt(os.Stdout, fmt.Sprintf("  %s Claude Code CLI? [Y/n] ", action))
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "" && answer != "y" && answer != "yes" {
			fmt.Println("  Skipped Claude Code CLI upgrade.")
			return nil
		}
	}

	runCLIInstall(info)
	reportCLIVersion(info)

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

// isHomebrewBinary reports whether the given path belongs to a Homebrew-managed installation.
func isHomebrewBinary(path string) bool {
	return strings.Contains(path, "/opt/homebrew/") ||
		strings.Contains(path, "/usr/local/Cellar/") ||
		strings.Contains(path, "/usr/local/opt/")
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
