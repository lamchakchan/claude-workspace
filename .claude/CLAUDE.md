# Project Instructions

## Project
Name: claude-workspace (Claude Code Platform Engineering Kit)
Purpose: Preconfigured AI agent platform for deploying Claude Code across teams with safe defaults, layered prompts, and multi-project support
Tech Stack: Go 1.24, Shell scripts, JSON, Markdown
Build: `go build -o claude-workspace .` or `make build`
Test: `go test ./...` or `make test`
Lint: `go vet ./...` or `make vet`

## Key Directories
- `_template/.claude/` - Embeddable assets overlaid into target projects (agents, skills, hooks, settings)
- `_template/.claude/agents/` - Five subagent definitions: planner, explorer, code-reviewer, test-runner, security-scanner
- `_template/.claude/skills/` - Four skill definitions: context-manager, onboarding, plan-and-execute, pr-workflow
- `_template/.claude/hooks/` - Four safety hooks: auto-format, block-dangerous-commands, enforce-branch-policy, validate-secrets
- `internal/attach/` - `attach` command: overlays template assets into a target project
- `internal/setup/` - `setup` command: installs Claude Code CLI and provisions API key
- `internal/sandbox/` - `sandbox` command: creates git worktrees for parallel Claude instances
- `internal/mcp/` - `mcp` command: manages MCP server configurations
- `internal/upgrade/` - `upgrade` command: upgrades claude-workspace and Claude Code CLI
- `internal/doctor/` - `doctor` command: validates platform configuration health
- `internal/platform/` - Shared utilities: FS abstraction, ExtractTo, file operations
- `scripts/` - Build, smoke-test, and dev-env scripts
- `docs/` - CLI reference, architecture, getting-started, and runbook

## Conventions
- Go: Standard gofmt/effective Go; each command lives in `internal/<command>/` with a `Run()` entrypoint
- Template assets are embedded via `assets.go` using `//go:embed _template` and exposed via `platform.FS`
- Shell scripts: `set -euo pipefail`, quoted variables, shellcheck-validated
- JSON config files use 2-space indentation, no trailing commas
- Markdown uses ATX headings and fenced code blocks with language tags
- Cross-platform binaries built for darwin/linux × amd64/arm64 via `make build-all`
- Version injected at build time via `-ldflags "-X main.version=..."`
- Pre-push validation: `make check` (vet + test + build)

## Important Files
- `main.go` - CLI entry point; command dispatch and embedded FS wiring
- `assets.go` - `//go:embed _template` directive that bundles all template assets
- `go.mod` - Module `github.com/lamchakchan/claude-workspace`, Go 1.24, minimal deps
- `Makefile` - All build, test, smoke-test, and dev-env targets
- `_template/.claude/settings.json` - Template Claude Code settings with safe defaults
- `_template/.mcp.json` - Template MCP server configurations
- `internal/platform/exec.go` - Shared exec/FS utilities used across all commands
- `internal/attach/attach.go` - Core logic for overlaying platform config into target projects
- `docs/CLI.md` - Full CLI reference for all commands and flags
- `docs/ARCHITECTURE.md` - Design philosophy, prompt layering, hook system, model strategy

## Important Notes
- The binary is named `claude-workspace`; the module path is `github.com/lamchakchan/claude-workspace`
- `attach --symlink` symlinks assets instead of copying, keeping projects in sync with template changes
- `attach --no-enrich` skips AI-powered CLAUDE.md enrichment (this onboarding flow)
- Safety hooks block `rm -rf`, force-push, secret patterns — do not bypass with `--no-verify`
- MCP auth flags (`--api-key`, `--bearer`, `--oauth`) use masked terminal input via `golang.org/x/term`
- Smoke tests require Docker or a VM; use `make smoke-test-fast` to skip Claude CLI installation
