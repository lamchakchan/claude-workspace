# Global Claude Code Instructions

## Identity
You are an AI coding agent operating within a governed platform environment.
Follow the platform conventions, use subagents for delegation, and plan before implementing.

## Defaults
- Always use TodoWrite for multi-step tasks
- Prefer Sonnet for coding, Opus for planning, Haiku for exploration
- Read files before modifying them
- Run tests after making changes
- Never commit secrets or credentials

## Code Quality Standards

### Universal Principles
- **Algorithmic complexity**: Choose optimal O() for the use case; prefer hash-based lookups over linear scans; avoid nested loops that can be flattened with better data structures
- **DRY**: Search for existing implementations before writing new code; extract shared logic when the same pattern appears 3+ times across files
- **Style consistency**: Match the existing codebase exactly — naming, structure, idioms, patterns; new code should read as if the same person wrote it
- **Resource management**: Prefer long-lived, shared, pooled resources (connections, clients, buffers); create once at a high level, pass down; never create per-request when a pool exists
- **Memory efficiency**: Avoid unnecessary allocations; pre-allocate when size is known; reuse buffers for repeated operations
- **Caching**: Compute/encode/decode at the highest level and pass results down; don't re-derive data that can be passed as a parameter; cache expensive computations when results are reused within scope
- **Proportionality**: Match optimization effort to the code's actual performance sensitivity; don't over-optimize cold paths

### Language-Specific Patterns

**Go:**
- `strings.Builder` over `+` concatenation; pre-allocate slices with `make([]T, 0, cap)`; pointer receivers for large structs
- Error wrapping: `%w` for chain-unwrapping, `%v` for opaque; `defer` for cleanup; check `Close()` errors on writers
- Lint: `golangci-lint run` if available, else `go vet`

**TypeScript/JavaScript:**
- `Map`/`Set` for lookups over object/array; `for...of` over `.forEach()` for early-exit
- Memoize with `useMemo`/`useCallback` only when deps actually change; avoid closures capturing loop vars
- Lint: `eslint` or `biome`; type-check with `tsc --noEmit`

**Python:**
- `dict`/`set` for O(1) lookups; generators over list comprehensions for large data; `__slots__` for many-instance classes
- Context managers (`with`) for cleanup; `functools.lru_cache` for pure function memoization
- Lint: `ruff` or `pylint`; type-check with `mypy`

**Java:**
- Use `HashMap`/`HashSet` for lookups; `StringBuilder` over `+` concatenation; streams for declarative collection processing
- Try-with-resources for all `AutoCloseable`; prefer `Optional` over null checks; immutable collections where possible
- Lint: `errorprone` or `spotbugs`; `checkstyle` for style

**.NET/C#:**
- `Dictionary`/`HashSet` for lookups; `StringBuilder` for string building; `Span<T>` for zero-allocation slicing
- `using`/`IDisposable` for resource cleanup; `async/await` over `.Result`; immutable records where appropriate
- Lint: Roslyn analyzers, `dotnet format`

**Erlang/Elixir:**
- Pattern matching over conditional chains; ETS tables for shared mutable state; avoid list operations on large datasets (use streams)
- OTP supervision trees for fault tolerance; prefer message passing over shared state
- Lint: `dialyzer` for type checking, `credo` (Elixir)

**Shell/Bash:**
- Quote all variables; `set -euo pipefail`; parameter expansion over external commands
- Cache command outputs in variables; avoid calling the same command twice

**C++:**
- RAII and smart pointers (`unique_ptr`/`shared_ptr`) over raw `new`/`delete`; `const` references for read-only params
- Move semantics for expensive-to-copy objects; `std::string_view` for non-owning string reads
- Lint: `clang-tidy` or `cppcheck`

**Ruby:**
- `Enumerable` methods over manual iteration; `Hash`/`Set` for O(1) lookups; `freeze` string literals
- `begin`/`rescue` at minimal scope; avoid `rescue Exception`; prefer keyword arguments for clarity
- Lint: `rubocop`; type-check with `sorbet` or `steep`

**PHP:**
- Type declarations on all function signatures; PDO prepared statements (no string interpolation in SQL)
- PSR-4 autoloading; `match` over `switch` (PHP 8+); named arguments for readability
- Lint: `phpstan` or `psalm`; style with `php-cs-fixer`

**Swift:**
- Value types (`struct`) over reference types when no identity needed; `guard` for early returns
- `Codable` for JSON; `async`/`await` for concurrency; structured concurrency with task groups
- Lint: `swiftlint`

**Kotlin:**
- Data classes for DTOs; `sealed class` for exhaustive `when`; coroutines over callbacks
- `?.let {}` over null checks; extension functions for utility; `Flow` for reactive streams
- Lint: `ktlint` or `detekt`

**Scala:**
- Immutable collections preferred; pattern matching over `isInstanceOf`; `case class` for value objects
- `for`-comprehension for monadic composition; `Option`/`Either` over null/exceptions
- Lint: `scalafmt`; `wartremover` for additional checks

## Git Conventions
- Work on feature branches, never main/master
- Commit messages: imperative mood, explain "why"
- Create PRs with clear descriptions

## MCP Tool Preferences

Prefer installed MCP tools over built-in Claude Code tools when both can satisfy the same request. MCP tools follow the `mcp__<server>__<tool>` naming pattern — identify them at runtime from the available tool list.

| Capability | Prefer MCP tools from providers like... | Over built-in... |
|---|---|---|
| Web search | brave, perplexity, tavily, exa, duckduckgo | `WebSearch` |
| Filesystem | filesystem | Bash file commands (cat, ls, find) |
| GitHub / VCS | github, gitlab, bitbucket | `gh` CLI via Bash |
| Observability | honeycomb, datadog, grafana, newrelic, sentry | (no built-in equivalent) |
| Persistent knowledge / memory | mcp-memory-libsql | (no built-in equivalent) |

If no MCP tool covers a capability, fall back to built-in tools normally. When multiple MCP tools could apply, choose the one whose description best matches the request (e.g., local vs. web search).

## Memory Strategy

Three memory layers, each for its right scope:

- **User CLAUDE.md** (`~/.claude/CLAUDE.md`): Rules and preferences for all projects. Always loaded.
- **Auto-memory** (`~/.claude/projects/<project>/memory/`): Project-specific facts Claude learns during work. Auto-loaded (first 200 lines). Use `/memory` to view or edit.
- **Memory MCP** (`mcp__mcp-memory-libsql__*`): Cross-project factual knowledge (not rules — those belong in CLAUDE.md). NOT auto-loaded. Inspect with `claude-workspace memory`.

**Session start rule**: Call `mcp__mcp-memory-libsql__read_graph` to load all stored cross-project memories. Use `read_graph` (not `search_nodes`) to avoid keyword-matching issues with the underlying FTS engine (no stemming — "preference" won't match "preferences").

**When saving to MCP memory:**
- One entity per topic, short kebab-case names (e.g., `go-conventions`, `git-workflow`)
- One fact per observation, key term first, under 100 chars
- Entity types: `preference` | `pattern` | `convention` | `tool-config` | `workflow`
- Only save cross-project facts here. Project-specific facts → auto-memory. Rules/instructions → CLAUDE.md.

See `docs/MEMORY.md` for the full reference including all layers, clearing procedures, and gitignore rules.
