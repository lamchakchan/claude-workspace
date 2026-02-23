package tools

import (
	"fmt"
	"os"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// Tool represents an installable system tool.
type Tool struct {
	Name       string
	Purpose    string
	Required   bool
	InstallCmd string                // manual install hint; auto-detected if empty
	CheckFn    func() bool           // nil → defaults to platform.Exists(Name)
	InstallFn  func() error          // nil → defaults to system package manager
	VersionFn  func() (string, error) // nil → no version reported
}

// IsInstalled reports whether the tool is available.
func (t Tool) IsInstalled() bool {
	if t.CheckFn != nil {
		return t.CheckFn()
	}
	return platform.Exists(t.Name)
}

// Install runs the tool's install function.
func (t Tool) Install() error {
	if t.InstallFn != nil {
		return t.InstallFn()
	}
	return platform.InstallSystemPackages(platform.DetectPackageManager(), []string{t.Name})
}

// Version returns the tool's version string.
func (t Tool) Version() (string, error) {
	if t.VersionFn != nil {
		return t.VersionFn()
	}
	return "", fmt.Errorf("no version function for %s", t.Name)
}

// InstallHint returns the manual install command for this tool.
func (t Tool) InstallHint() string {
	if t.InstallCmd != "" {
		return t.InstallCmd
	}
	return platform.InstallHintForPM(platform.DetectPackageManager(), t.Name)
}

// CheckAndInstall checks which tools are installed, attempts to install missing ones,
// and reports results. Returns lists of installed and failed tool names.
func CheckAndInstall(tools []Tool) (installed, failed []string) {
	var missing []Tool
	var found []string

	for _, t := range tools {
		if t.IsInstalled() {
			found = append(found, t.Name)
		} else {
			missing = append(missing, t)
		}
	}

	if len(found) > 0 {
		platform.PrintSuccess(os.Stdout, fmt.Sprintf("Found: %s", strings.Join(found, ", ")))
	}

	if len(missing) == 0 {
		fmt.Println("  All optional tools are available.")
		return found, nil
	}

	// Attempt to install missing tools
	pm := platform.DetectPackageManager()

	// Batch install system-package tools (those without custom InstallFn)
	if pm != platform.PMNone {
		var sysNames []string
		for _, t := range missing {
			if t.InstallFn == nil {
				sysNames = append(sysNames, t.Name)
			}
		}
		if len(sysNames) > 0 {
			fmt.Printf("\n  Attempting to install: %s\n", strings.Join(sysNames, ", "))
			if err := platform.InstallSystemPackages(pm, sysNames); err == nil {
				fmt.Printf("  Installed successfully via %s.\n", platform.PMLabel(pm))
			}
		}
	}

	// Run custom installs for tools with InstallFn
	for _, t := range missing {
		if t.InstallFn != nil && !t.IsInstalled() {
			fmt.Printf("\n  Installing %s (%s)...\n", t.Name, t.Purpose)
			if err := t.InstallFn(); err == nil {
				platform.PrintOK(os.Stdout, fmt.Sprintf("Installed %s", t.Name))
			} else {
				platform.PrintWarn(os.Stdout, fmt.Sprintf("Failed to install %s", t.Name))
			}
		}
	}

	// Re-check what's still missing
	var stillMissing []Tool
	for _, t := range missing {
		if t.IsInstalled() {
			installed = append(installed, t.Name)
		} else {
			stillMissing = append(stillMissing, t)
			failed = append(failed, t.Name)
		}
	}

	if len(stillMissing) > 0 {
		fmt.Println("\n  Optional tools not found (not required, but useful):")
		for _, t := range stillMissing {
			fmt.Printf("    - %s: %s\n", platform.Bold(t.Name), t.Purpose)
			platform.PrintCommand(os.Stdout, t.InstallHint())
		}
	} else {
		fmt.Println("  All optional tools are available.")
	}

	installed = append(found, installed...)
	return installed, failed
}
