# Project Instructions

## Project
Name: claude-workspace
Purpose: A preconfigured, batteries-included platform for deploying Claude Code AI agents across organizations
Tech Stack: Go 1.24, embed FS, shell scripts, CUE (template linting)
Build: `make build` (or `go build ./...` for quick single-platform build)
Test: `go test ./...`
Lint: `go vet ./...` and `make lint` (CUE-based template validation)

## Key Directories
- internal/            - All application packages (one package per CLI command)
- internal/platform/   - Shared utilities: file ops, color output, JSON helpers, exec, env detection, CLAUDE.md generation, package manager detection
- internal/attach/     - `attach` command: overlays platform config onto target projects
- internal/enrich/     - `enrich` command: re-generate .claude/CLAUDE.md with AI analysis
- internal/setup/      - `setup` command: first-time setup, API key provisioning, npm detection
- internal/mcp/        - `mcp` command: add/list/remote MCP server configurations
- internal/sandbox/    - `sandbox` command: git worktree creation for parallel development
- internal/tools/      - Tool registry: defines required (claude, node) and optional (engram, shellcheck, jq, prettier, tmux) tools
- internal/doctor/     - `doctor` command: health checks for platform configuration
- internal/upgrade/    - `upgrade` command: self-update via GitHub releases
- internal/sessions/   - `sessions` command: browse and review session prompts
- internal/memory/     - `memory` command: inspect and manage memory layers (show, export, import)
- internal/cost/       - `cost` command: usage and cost reporting via ccusage
- internal/statusline/ - `statusline` command: configure Claude Code statusline display
- _template/project/   - Embedded assets copied into target projects by `attach` (agents, skills, hooks, settings)
- _template/global/    - Embedded global-level assets (global CLAUDE.md, global settings)
- docs/                - Documentation: architecture, CLI reference, getting started, config, MCP configs, runbook, memory
- scripts/             - Shell scripts: smoke tests, dev environment management, CI helpers, template linting, dependency auto-install, shared libraries
- lint/                - CUE schemas for validating template JSON files (agents, skills, settings, MCP configs)
- plans/               - Implementation plans directory (created by attach in target projects)

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
- Only one external dependency: `golang.org/x/term` (for masked terminal input)
- Scripts share common functions via `lib.sh`, `lib-phases.sh`, and `lib-provision.sh` in `scripts/`
- `make dep` auto-installs Go and CUE if missing; individual targets (test, vet, lint, build) also ensure their deps before running

## Important Files
- `main.go` - CLI entry point, command routing, embedded FS wiring
- `assets.go` - `//go:embed` directive that bundles `_template/` into the binary
- `internal/platform/assets.go` - Asset extraction (copy/symlink) and embedded FS access
- `internal/platform/claudemd.go` - CLAUDE.md scaffold generation, tech stack detection, and AI enrichment via claude CLI
- `internal/platform/exec.go` - Shell command execution helpers used across packages
- `internal/platform/fs.go` - File system utilities (FileExists, etc.)
- `internal/platform/pkgmgr.go` - Package manager detection (npm, bun, etc.)
- `internal/platform/env.go` - Environment variable helpers
- `internal/attach/attach.go` - Core logic for overlaying platform config onto projects
- `internal/enrich/enrich.go` - Standalone CLAUDE.md enrichment command (scaffold + AI analysis)
- `internal/tools/registry.go` - Tool registry defining required (claude, node) vs optional dependencies
- `internal/setup/setup.go` - First-time setup flow including Claude CLI and API key provisioning
- `Makefile` - Build targets, cross-compilation, smoke tests, dev environment management
- `install.sh` - Curl-pipe installer for end users
- `scripts/install-deps.sh` - Auto-detection and installation of Go and CUE toolchains (reads go.mod for version)

## Important Notes
- The `_template/project/` directory must stay in sync with the root `.claude/` folder; changes to agents, skills, hooks, or settings in one must be mirrored to the other (exception: auto-memory files under `.claude/` are NOT mirrored)
- The `_template/global/` directory contains global-level assets (e.g., global CLAUDE.md, global settings) embedded separately from project assets
- `make build` produces four cross-platform binaries and creates a local symlink; `make check` runs vet + test + lint + build as pre-push validation
- `make smoke-test` runs end-to-end tests in a temp directory; `--docker` and `--vm` variants available via `scripts/dev-env.sh`
- The binary embeds all template assets at compile time — changes to `_template/` require rebuilding
- AI enrichment (`enrich` command and `attach` without `--no-enrich`) shells out to `claude -p --model opus` to analyze the target project and generate a detailed CLAUDE.md; requires a valid API key
- The `lint/` directory contains CUE schemas that validate template JSON files; run `make lint` or `bash scripts/lint-templates.sh`
- The `upgrade` package has three files: `upgrade.go` (orchestration), `selfupdate.go` (binary self-update), `github.go` (GitHub release API)
- The `memory` package has three files: `memory.go` (CLI routing), `layers.go` (layer inspection), `export.go` (JSON export/import)
