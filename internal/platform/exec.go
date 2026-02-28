package platform

import (
	"bytes"
	"context"
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

// OutputContext is like Output but accepts a context for cancellation.
func OutputContext(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
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

// RunDirWithStdin executes a command in a specific directory with stdin from a string.
// Returns trimmed stdout. Stderr is discarded.
func RunDirWithStdin(ctx context.Context, dir string, stdin string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(stdin)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

// RunDirWithStdinCapture executes a command in a specific directory with stdin from a string.
// unsetEnv lists environment variable names to strip from the inherited environment.
// Returns trimmed stdout and stderr separately.
func RunDirWithStdinCapture(ctx context.Context, dir string, stdin string, unsetEnv []string, name string, args ...string) (stdout, stderr string, err error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(stdin)
	if len(unsetEnv) > 0 {
		strip := make(map[string]bool, len(unsetEnv))
		for _, k := range unsetEnv {
			strip[k] = true
		}
		filtered := make([]string, 0, len(os.Environ()))
		for _, e := range os.Environ() {
			key, _, _ := strings.Cut(e, "=")
			if !strip[key] {
				filtered = append(filtered, e)
			}
		}
		cmd.Env = filtered
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return strings.TrimSpace(outBuf.String()), strings.TrimSpace(errBuf.String()), err
}

// RunWithSpinner runs a command with stdin/stderr inherited and shows a spinner on stderr
// until the subprocess produces its first byte of stdout, then streams the rest through.
func RunWithSpinner(msg string, name string, args ...string) error {
	pr, pw, err := os.Pipe()
	if err != nil {
		return Run(name, args...)
	}

	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = pw

	spinner := StartSpinner(os.Stderr, msg)
	if err := cmd.Start(); err != nil {
		spinner.Stop()
		pr.Close()
		pw.Close()
		return err
	}
	pw.Close()

	spinnerStopped := false
	buf := make([]byte, 4096)
	for {
		n, readErr := pr.Read(buf)
		if n > 0 {
			if !spinnerStopped {
				spinner.Stop()
				spinnerStopped = true
			}
			os.Stdout.Write(buf[:n])
		}
		if readErr != nil {
			break
		}
	}
	if !spinnerStopped {
		spinner.Stop()
	}
	pr.Close()
	return cmd.Wait()
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
