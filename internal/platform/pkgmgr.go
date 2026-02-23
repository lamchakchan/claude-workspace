package platform

import (
	"fmt"
	"runtime"
	"strings"
)

// PackageManager represents a system package manager.
type PackageManager int

const (
	PMNone   PackageManager = iota
	PMBrew                  // macOS (Homebrew)
	PMApt                   // Debian, Ubuntu
	PMDnf                   // Fedora, RHEL, CentOS Stream
	PMPacman                // Arch, Manjaro
	PMApk                   // Alpine
)

// DetectPackageManager returns the detected system package manager.
func DetectPackageManager() PackageManager {
	if runtime.GOOS == "darwin" && Exists("brew") {
		return PMBrew
	}
	if Exists("apt-get") {
		return PMApt
	}
	if Exists("dnf") {
		return PMDnf
	}
	if Exists("pacman") {
		return PMPacman
	}
	if Exists("apk") {
		return PMApk
	}
	return PMNone
}

// InstallSystemPackages batch-installs packages via the given package manager.
func InstallSystemPackages(pm PackageManager, names []string) error {
	if len(names) == 0 {
		return nil
	}
	hasSudo := Exists("sudo")
	switch pm {
	case PMBrew:
		args := append([]string{"install"}, names...)
		return RunQuiet("brew", args...)
	case PMApt:
		if hasSudo {
			args := append([]string{"apt-get", "install", "-y"}, names...)
			return RunQuiet("sudo", args...)
		}
		args := append([]string{"install", "-y"}, names...)
		return RunQuiet("apt-get", args...)
	case PMDnf:
		if hasSudo {
			args := append([]string{"dnf", "install", "-y"}, names...)
			return RunQuiet("sudo", args...)
		}
		args := append([]string{"install", "-y"}, names...)
		return RunQuiet("dnf", args...)
	case PMPacman:
		if hasSudo {
			args := append([]string{"pacman", "-S", "--noconfirm"}, names...)
			return RunQuiet("sudo", args...)
		}
		args := append([]string{"-S", "--noconfirm"}, names...)
		return RunQuiet("pacman", args...)
	case PMApk:
		args := append([]string{"add"}, names...)
		return RunQuiet("apk", args...)
	default:
		return fmt.Errorf("no package manager detected")
	}
}

// InstallHintForPM returns the appropriate install command string for user display.
func InstallHintForPM(pm PackageManager, name string) string {
	switch pm {
	case PMBrew:
		return "brew install " + name
	case PMApt:
		if Exists("sudo") {
			return "sudo apt-get install -y " + name
		}
		return "apt-get install -y " + name
	case PMDnf:
		if Exists("sudo") {
			return "sudo dnf install -y " + name
		}
		return "dnf install -y " + name
	case PMPacman:
		if Exists("sudo") {
			return "sudo pacman -S --noconfirm " + name
		}
		return "pacman -S --noconfirm " + name
	case PMApk:
		return "apk add " + name
	default:
		if runtime.GOOS == "darwin" {
			return "brew install " + name + " (macOS) / apt install " + name + " (Linux)"
		}
		return "apt install " + name
	}
}

// String returns the package manager name.
func (pm PackageManager) String() string {
	names := []string{"none", "brew", "apt", "dnf", "pacman", "apk"}
	if int(pm) < len(names) {
		return names[pm]
	}
	return fmt.Sprintf("PackageManager(%d)", pm)
}

// PMInstallNames returns the package names used by a specific package manager.
// Some package managers use different package names (e.g., "nodejs" vs "node").
// This is a convenience for callers that need PM-specific name mapping.
func PMInstallNames(pm PackageManager, names []string) []string {
	// Default: return as-is. Callers can override per-tool if needed.
	result := make([]string, len(names))
	copy(result, names)
	return result
}

// PMLabel returns a human-readable label for the package manager (e.g., "brew", "apt").
func PMLabel(pm PackageManager) string {
	switch pm {
	case PMBrew:
		return "brew"
	case PMApt:
		return "apt"
	case PMDnf:
		return "dnf"
	case PMPacman:
		return "pacman"
	case PMApk:
		return "apk"
	default:
		return "system package manager"
	}
}

// JoinNames joins package names for display.
func JoinNames(names []string) string {
	return strings.Join(names, ", ")
}
