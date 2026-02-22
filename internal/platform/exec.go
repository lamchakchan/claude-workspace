package platform

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Run executes a command with stdin/stdout/stderr inherited.
func Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunDir executes a command in a specific directory with inherited I/O.
func RunDir(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunQuiet executes a command and discards stdout/stderr.
func RunQuiet(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// RunQuietWithEnv executes a command with extra environment variables, discarding output.
func RunQuietWithEnv(extraEnv []string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// RunQuietDir executes a command in a specific directory, discarding output.
func RunQuietDir(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// Output executes a command and returns its stdout as a trimmed string.
func Output(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

// OutputDir executes a command in a specific directory and returns its stdout.
func OutputDir(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

// Exists checks if a command exists in PATH.
func Exists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// RunSpawn runs a command with full I/O passthrough and returns the exit code.
func RunSpawn(name string, args ...string) (int, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return -1, fmt.Errorf("running %s: %w", name, err)
	}
	return 0, nil
}
