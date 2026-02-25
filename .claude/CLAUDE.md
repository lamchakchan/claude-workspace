# Project Instructions

## Project
Name: claude-workspace
Purpose: A preconfigured, batteries-included platform for deploying Claude Code AI agents across organizations
Tech Stack: Go 1.24
Build: `go build ./...` or `make build-all` (cross-compiles darwin/linux, amd64/arm64)
Test: `go test ./...`
Lint: `go vet ./...`

## Key Directories
- internal/          - Core Go packages, one per CLI subcommand
- internal/platform/ - Shared utilities (exec, filesystem, JSON, color output, env detection)
- internal/attach/   - `attach` command: overlays platform config onto target projects
- internal/setup/    - `setup` command: first-time setup and API key provisioning
- internal/mcp/      - `mcp` command: add/list/remote MCP server management
- internal/sandbox/  - `sandbox` command: git worktree creation for parallel dev
- internal/doctor/   - `doctor` command: platform health checks
- internal/upgrade/  - `upgrade` command: self-update via GitHub releases
- internal/cost/     - `cost` command: usage and cost reporting via ccusage
- internal/sessions/ - `sessions` command: browse and review session prompts
- internal/statusline/ - `statusline` command: configure Claude Code statusline
- internal/tools/    - Tool dependency registry (node, jq, prettier, shellcheck, etc.)
- _template/         - Embedded assets (.claude config, .mcp.json) deployed by `attach`
- .claude/agents/    - Subagent definitions (planner, explorer, code-reviewer, etc.)
- .claude/skills/    - Skill definitions (onboarding, pr-workflow, plan-and-execute, etc.)
- .claude/hooks/     - Safety hook scripts (block dangerous commands, validate secrets, etc.)
- docs/              - User-facing documentation (getting started, architecture, CLI reference, runbook)
- scripts/           - Dev tooling (smoke-test.sh, dev-env.sh, lib.sh)
- plans/             - Implementation plan documents

## Conventions
- One package per CLI subcommand under `internal/`, each with a `Run()` entry point
- Command dispatch via switch statement in `main.go` (no CLI framework)
- Shared helpers live in `internal/platform/` — exec wrappers, file ops, JSON helpers, color output
- Template assets in `_template/` are embedded via `//go:embed all:_template` in `assets.go`
- Error handling: return `fmt.Errorf("context: %w", err)` with wrapped errors; print to stderr and `os.Exit(1)` in main
- Test files co-located with source: `foo.go` / `foo_test.go`
- Version injected at build time via `-ldflags "-X main.version=$(VERSION)"`
- Cross-compilation for 4 targets: darwin/linux × amd64/arm64
- Color output via `internal/platform/color.go` helper functions (PrintBanner, PrintStep, PrintSuccess, etc.)
- No external CLI framework or dependency injection; minimal dependencies (only `golang.org/x/term`)

## Important Files
- main.go - CLI entry point and command dispatch
- assets.go - Embeds `_template/` directory into the binary
- Makefile - Build, test, cross-compile, smoke test, and dev environment targets
- internal/platform/exec.go - Command execution helpers used throughout
- internal/platform/assets.go - Embedded filesystem extraction and asset management
- internal/attach/attach.go - Core attach logic including CLAUDE.md enrichment via Claude API
- internal/setup/setup.go - First-time setup workflow
- _template/.claude/settings.json - Default Claude Code settings deployed to projects
- _template/.mcp.json - Default MCP server configuration deployed to projects
- install.sh - Curl-pipe installer for end users

## Important Notes
- The `_template/` directory is embedded into the binary at compile time — changes to template files require a rebuild
- `make build-all` (or `make build`) cross-compiles all 4 platform binaries and creates a local symlink
- `make check` runs vet + test + build as a pre-push validation gate
- The `attach --symlink` mode extracts assets to `~/.claude-workspace/assets/` and symlinks them, keeping multiple projects in sync
- CLAUDE.md enrichment during `attach` spawns a `claude -p` subprocess with Opus model and 180s timeout
- `go vet ./...` is the only lint tool configured (no golangci-lint)
- Releases are managed via GoReleaser (`.goreleaser.yaml`)
- Dev environments can be created in Docker or VMs via `scripts/dev-env.sh`
