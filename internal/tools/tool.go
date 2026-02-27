// Package tools provides a registry of required and optional system tools
// with detection, installation, and version reporting capabilities.
package tools

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lamchakchan/claude-workspace/internal/platform"
)

// Tool represents an installable system tool.
type Tool struct {
	Name       string
	Purpose    string
	Required   bool
	InstallCmd string                 // manual install hint; auto-detected if empty
	CheckFn    func() bool            // nil → defaults to platform.Exists(Name)
	InstallFn  func() error           // nil → defaults to system package manager
	VersionFn  func() (string, error) // nil → no version reported
}

// IsInstalled reports whether the tool is available.
func (t *Tool) IsInstalled() bool {
	if t.CheckFn != nil {
		return t.CheckFn()
	}
	return platform.Exists(t.Name)
}

// Install runs the tool's install function.
func (t *Tool) Install() error {
	if t.InstallFn != nil {
		return t.InstallFn()
	}
	return platform.InstallSystemPackages(platform.DetectPackageManager(), []string{t.Name})
}

// Version returns the tool's version string.
func (t *Tool) Version() (string, error) {
	if t.VersionFn != nil {
		return t.VersionFn()
	}
	return "", fmt.Errorf("no version function for %s", t.Name)
}

// InstallHint returns the manual install command for this tool.
func (t *Tool) InstallHint() string {
	if t.InstallCmd != "" {
		return t.InstallCmd
	}
	return platform.InstallHintForPM(platform.DetectPackageManager(), t.Name)
}

// CheckAndInstall checks which tools are installed, attempts to install missing ones,
// and reports results. Returns lists of installed and failed tool names.
func CheckAndInstall(toolList []Tool) (installed, failed []string) {
	return checkAndInstallTo(os.Stdout, toolList)
}

// CheckAndInstallTo is like CheckAndInstall but writes output to w instead of os.Stdout.
func CheckAndInstallTo(w io.Writer, toolList []Tool) (installed, failed []string) {
	return checkAndInstallTo(w, toolList)
}

func checkAndInstallTo(w io.Writer, toolList []Tool) (installed, failed []string) {
	found, missing := partitionTools(toolList)

	if len(found) > 0 {
		platform.PrintSuccess(w, fmt.Sprintf("Found: %s", strings.Join(found, ", ")))
	}

	if len(missing) == 0 {
		fmt.Fprintln(w, "  All optional tools are available.")
		return found, nil
	}

	attemptInstallsTo(w, missing)

	nowInstalled, stillMissing := partitionTools(missing)
	reportMissingTo(w, stillMissing)

	allInstalled := make([]string, 0, len(found)+len(nowInstalled))
	allInstalled = append(allInstalled, found...)
	allInstalled = append(allInstalled, nowInstalled...)
	failed = make([]string, 0, len(stillMissing))
	for _, t := range stillMissing {
		failed = append(failed, t.Name)
	}
	return allInstalled, failed
}

// partitionTools splits tools into found (name strings) and missing (Tool slices).
func partitionTools(toolList []Tool) (found []string, missing []Tool) {
	for _, t := range toolList {
		if t.IsInstalled() {
			found = append(found, t.Name)
		} else {
			missing = append(missing, t)
		}
	}
	return found, missing
}

// attemptInstalls tries to install missing tools via system packages and custom installers.
func attemptInstalls(missing []Tool) {
	attemptInstallsTo(os.Stdout, missing)
}

func attemptInstallsTo(w io.Writer, missing []Tool) {
	pm := platform.DetectPackageManager()

	if pm != platform.PMNone {
		var sysNames []string
		for _, t := range missing {
			if t.InstallFn == nil {
				sysNames = append(sysNames, t.Name)
			}
		}
		if len(sysNames) > 0 {
			fmt.Fprintf(w, "\n  Attempting to install: %s\n", strings.Join(sysNames, ", "))
			if err := platform.InstallSystemPackages(pm, sysNames); err == nil {
				fmt.Fprintf(w, "  Installed successfully via %s.\n", platform.PMLabel(pm))
			}
		}
	}

	for _, t := range missing {
		if t.InstallFn != nil && !t.IsInstalled() {
			fmt.Fprintf(w, "\n  Installing %s (%s)...\n", t.Name, t.Purpose)
			if err := t.InstallFn(); err == nil {
				platform.PrintOK(w, fmt.Sprintf("Installed %s", t.Name))
			} else {
				platform.PrintWarn(w, fmt.Sprintf("Failed to install %s", t.Name))
			}
		}
	}
}

// reportMissing prints instructions for tools that could not be installed.
func reportMissing(stillMissing []Tool) {
	reportMissingTo(os.Stdout, stillMissing)
}

func reportMissingTo(w io.Writer, stillMissing []Tool) {
	if len(stillMissing) > 0 {
		fmt.Fprintln(w, "\n  Optional tools not found (not required, but useful):")
		for _, t := range stillMissing {
			fmt.Fprintf(w, "    - %s: %s\n", platform.Bold(t.Name), t.Purpose)
			platform.PrintCommand(w, t.InstallHint())
		}
	} else {
		fmt.Fprintln(w, "  All optional tools are available.")
	}
}
