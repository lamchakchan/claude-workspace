# Project Instructions

## Project
Name: claude-workspace
Purpose: A platform engineering kit for deploying Claude Code AI agents across organizations
Tech Stack: Go 1.24, Charm TUI (bubbletea/bubbles/lipgloss), embed FS
Build: `make build` (cross-compiles darwin/linux, amd64/arm64)
Test: `go test ./...`
Lint: `go vet ./...` and `golangci-lint run ./...` and `make lint` (CUE template validation)

## Key Directories
- internal/          - All application packages (one per CLI subcommand)
- internal/platform/ - Shared utility layer (exec, fs, json, color, env, assets, pkgmgr, claudemd)
- internal/tui/      - Interactive TUI menu (bubbletea-based)
- internal/tools/    - Tool registry with detection for Node, Claude CLI, etc.
- _template/         - Embedded templates (source of truth for attach/setup output)
- docs/              - User documentation (architecture, CLI reference, config, runbook)
- scripts/           - Build, test, CI, and dev environment scripts
- lint/              - CUE schemas for validating template files (agents, skills, settings, MCP)
- plans/             - Implementation plan documents

## Conventions
- Each `internal/` subpackage has one main file named after the package with a public `Run()` entry point
- Packages may expose `RunTo(w io.Writer)` for testability
- Error wrapping uses `fmt.Errorf("context: %w", err)` with lowercase gerund context phrases
- Errors are returned upward; only `main.go` prints to stderr and calls `os.Exit`
- No `init()` functions; all initialization is explicit in `main()`
- Imports organized in 3 groups: stdlib, external deps, internal packages (alpha-sorted within each)
- Table-driven tests using `tt` loop variable, standard library only (no testify)
- Test naming: `TestFunc` for happy path, `TestFunc_EdgeCase` with underscore-separated variants
- Pre-allocate slices with `make([]T, 0, cap)` when size is estimable
- Use `strings.Builder` for string concatenation
- Minimal external dependencies by design — avoid adding new ones
- `_template/` must stay in sync with root `.claude/` and `.mcp.json` (except auto-memory dirs)
- Cross-platform: always build/test for darwin+linux, amd64+arm64
- Package manager agnostic: support brew/apt/dnf/pacman/apk via `platform.DetectPackageManager()`

## Important Files
- main.go            - CLI entrypoint, command routing, embedded FS wiring
- assets.go          - `//go:embed all:_template` directive for embedded templates
- Makefile           - Build, test, lint, smoke-test, dev environment targets
- internal/platform/exec.go     - Command execution helpers (Run, Output, RunWithSpinner)
- internal/platform/fs.go       - Filesystem utilities (CopyFile, WalkFiles, SymlinkFile)
- internal/platform/claudemd.go - Project detection and CLAUDE.md scaffold generation
- internal/platform/color.go    - ANSI color and structured print helpers (PrintBanner, PrintStep, etc.)
- internal/platform/assets.go   - Embedded FS extraction (ExtractTo, ReadAsset)
- internal/attach/attach.go     - Core `attach` command (copies/symlinks templates to target project)
- internal/tools/registry.go    - Tool registry pattern with Required/Optional/All accessors

## Important Notes
- Version is injected via `-ldflags "-X main.version=$(VERSION)"` at build time
- Embedded FS vars `platform.FS` and `platform.GlobalFS` are set by `main.go` before any command runs
- Print helpers in `platform/color.go` take `io.Writer` as first param for testability
- The `tools/` package uses a registry pattern: individual files (claude.go, node.go) return `Tool` structs
- `make check` runs full pre-push validation: vet + test + lint + golint + build
- `make smoke-test` runs integration tests; `--docker` variant tests in containers
