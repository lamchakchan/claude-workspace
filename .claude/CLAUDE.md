# Project Instructions

## Project
Name: claude-workspace
Purpose: A preconfigured, batteries-included platform for deploying Claude Code AI agents across organizations
Tech Stack: Go 1.24, embed FS, shell scripts
Build: `make build` (or `go build ./...` for quick single-platform build)
Test: `go test ./...`
Lint: `go vet ./...`

## Key Directories
- internal/          - All application packages (one package per CLI command)
- internal/platform/ - Shared utilities: file ops, color output, JSON helpers, exec, embedded asset extraction
- internal/attach/   - `attach` command: overlays platform config onto target projects
- internal/setup/    - `setup` command: first-time setup, API key provisioning, npm detection
- internal/mcp/      - `mcp` command: add/list/remote MCP server configurations
- internal/sandbox/  - `sandbox` command: git worktree creation for parallel development
- internal/tools/    - Tool registry: defines required (claude, node, engram) and optional (shellcheck, jq, prettier, tmux) tools
- internal/doctor/   - `doctor` command: health checks for platform configuration
- internal/upgrade/  - `upgrade` command: self-update via GitHub releases
- internal/sessions/ - `sessions` command: browse and review session prompts
- internal/memory/   - `memory` command: inspect and manage memory layers
- internal/cost/     - `cost` command: usage and cost reporting via ccusage
- internal/statusline/ - `statusline` command: configure Claude Code statusline display
- _template/project/ - Embedded assets copied into target projects by `attach` (agents, skills, hooks, settings)
- _template/global/  - Embedded global-level assets (global CLAUDE.md)
- docs/              - Documentation: architecture, CLI reference, getting started, MCP configs, runbook, memory
- scripts/           - Shell scripts: smoke tests, dev environment management, CI helpers
- plans/             - Implementation plans directory (created by attach in target projects)

## Conventions
- One package per CLI subcommand under `internal/` with a `Run()` entry point
- Each package has a corresponding `_test.go` file
- Version injected via `-ldflags "-X main.version=$(VERSION)"` at build time
- Embedded filesystem via `//go:embed all:_template` in `assets.go`; split into `platform.FS` (project) and `platform.GlobalFS` (global) in `main.go`
- CLI argument parsing is manual (switch/case in `main.go`), no external flag library
- Error handling: return errors up to `main()`, which prints to stderr and exits with code 1
- `internal/platform/` is the shared utility layer; other packages import it but not each other
- Cross-platform builds: darwin/arm64, darwin/amd64, linux/arm64, linux/amd64 via Makefile
- Shell scripts (.sh) get 0755 permissions when extracted; all other files get 0644

## Important Files
- `main.go` - CLI entry point, command routing, embedded FS wiring
- `assets.go` - `//go:embed` directive that bundles `_template/` into the binary
- `internal/platform/assets.go` - Asset extraction (copy/symlink) and embedded FS access
- `internal/platform/exec.go` - Shell command execution helpers used across packages
- `internal/platform/fs.go` - File system utilities (FileExists, etc.)
- `internal/attach/attach.go` - Core logic for overlaying platform config onto projects
- `internal/tools/registry.go` - Tool registry defining required vs optional dependencies
- `internal/setup/setup.go` - First-time setup flow including Claude CLI and API key provisioning
- `Makefile` - Build targets, cross-compilation, smoke tests, dev environment management
- `install.sh` - Curl-pipe installer for end users

## Important Notes
- The `_template/project/` directory must stay in sync with the root `.claude/` folder; changes to agents, skills, hooks, or settings in one must be mirrored to the other (exception: auto-memory files under `.claude/` are NOT mirrored)
- The `_template/global/` directory contains global-level assets (e.g., global CLAUDE.md) embedded separately from project assets
- `make build` produces four cross-platform binaries and creates a local symlink; `make check` runs vet + test + build as pre-push validation
- `make smoke-test` runs end-to-end tests in a temp directory; `--docker` and `--vm` variants available via `scripts/dev-env.sh`
- The binary embeds all template assets at compile time — changes to `_template/` require rebuilding
