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

## Team Execution

The platform supports team-based parallel execution via Claude Code's agent teams feature.

### When to create a team
- The task has 3+ implementation phases and would benefit from structured tracking
- Multiple independent workstreams can proceed in parallel on isolated files
- Complex sequential work benefits from automated verification hooks between phases (TaskCompleted runs tests on each phase completion)
- The user explicitly asks to "run this in parallel", "use a team", or "use agents"

### Execution modes
- **Sequential** (default): No team. Use for simple, linear tasks or when phases overlap files.
- **Solo team**: You create a team for yourself using `TeamCreate`. One task per phase via `TaskCreate`. You execute phases sequentially, marking each task completed. Benefits: TaskCompleted hooks run tests automatically between phases; TeammateIdle hooks provide nudges; structured progress tracking via `TaskList`. Use for complex sequential work that benefits from automated gates.
- **Multi-agent team (simple)**: You create the team with `TeamCreate`, spawn 1-2 teammates with the `Agent` tool (set `team_name` and `name`), assign tasks, and monitor directly. Use when 2 phases can run in parallel on isolated files.
- **Multi-agent team (complex)**: Delegate to the `team-lead` agent for 3+ teammates or multi-phase dependency graphs with phase transitions requiring verification.

### Key tools
- `TeamCreate` — create a new team with task list
- `TaskCreate` / `TaskUpdate` / `TaskList` — manage tasks within a team
- `Agent` tool with `team_name` and `name` params — spawn teammates
- `SendMessage` — communicate with teammates
- `TeamDelete` — clean up team after completion

### Hooks (configured in settings.json)
- **TaskCompleted** (`verify-task-completed.sh`): Runs project tests before allowing task completion
- **TeammateIdle** (`check-teammate-idle.sh`): Nudges idle teammates that have in-progress tasks

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
