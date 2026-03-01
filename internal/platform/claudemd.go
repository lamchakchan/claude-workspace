package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// jsDep maps a package.json dependency name to a human-readable framework name.
type jsDep struct {
	dep  string
	name string
}

// frameworkDeps are exclusive front-end frameworks (first match wins).
var frameworkDeps = []jsDep{
	{dep: "next", name: "Next.js"},
	{dep: "react", name: "React"},
	{dep: "vue", name: "Vue"},
	{dep: "@angular/core", name: "Angular"},
	{dep: "svelte", name: "Svelte"},
}

// addonDeps are additive dependencies (all matches are included).
var addonDeps = []jsDep{
	{dep: "express", name: "Express"},
	{dep: "fastify", name: "Fastify"},
	{dep: "hono", name: "Hono"},
	{dep: "typescript", name: "TypeScript"},
	{dep: "prisma", name: "Prisma"},
	{dep: "@prisma/client", name: "Prisma"},
	{dep: "drizzle", name: "Drizzle"},
}

// DetectJSTechStack scans package.json deps for known frameworks.
func DetectJSTechStack(pkg map[string]json.RawMessage) string {
	allDeps := collectJSDeps(pkg)

	var stack []string

	// Exclusive framework detection: first match wins
	for _, fw := range frameworkDeps {
		if allDeps[fw.dep] {
			stack = append(stack, fw.name)
			break
		}
	}

	// Additive dependency detection: all matches included
	seen := make(map[string]bool, len(addonDeps))
	for _, addon := range addonDeps {
		if allDeps[addon.dep] && !seen[addon.name] {
			stack = append(stack, addon.name)
			seen[addon.name] = true
		}
	}

	if len(stack) > 0 {
		return strings.Join(stack, ", ")
	}
	return "Node.js"
}

// collectJSDeps merges dependencies and devDependencies from a parsed package.json.
func collectJSDeps(pkg map[string]json.RawMessage) map[string]bool {
	allDeps := make(map[string]bool)
	for _, key := range []string{"dependencies", "devDependencies"} {
		if raw, ok := pkg[key]; ok {
			var deps map[string]string
			if json.Unmarshal(raw, &deps) == nil {
				for k := range deps {
					allDeps[k] = true
				}
			}
		}
	}
	return allDeps
}

// hasCppSources checks if C++ source files exist in the project root or src/ directory.
func hasCppSources(projectDir string) bool {
	for _, pattern := range []string{"*.cpp", "*.cc", "*.cxx", "src/*.cpp", "src/*.cc", "src/*.cxx"} {
		matches, _ := filepath.Glob(filepath.Join(projectDir, pattern))
		if len(matches) > 0 {
			return true
		}
	}
	return false
}

// hasKotlinSources checks if Kotlin source files exist in the project.
func hasKotlinSources(projectDir string) bool {
	for _, pattern := range []string{"*.kt", "src/**/*.kt", "src/*.kt"} {
		matches, _ := filepath.Glob(filepath.Join(projectDir, pattern))
		if len(matches) > 0 {
			return true
		}
	}
	return false
}

// hasGlobMatch checks if any file matches the given glob pattern in the project directory.
func hasGlobMatch(projectDir, pattern string) bool {
	matches, _ := filepath.Glob(filepath.Join(projectDir, pattern))
	return len(matches) > 0
}

// fileContains checks if a file contains a given substring.
func fileContains(path, substr string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), substr)
}

// detectSpringBoot checks if a Maven or Gradle project uses Spring Boot.
func detectSpringBoot(projectDir string) bool {
	if FileExists(filepath.Join(projectDir, "pom.xml")) && fileContains(filepath.Join(projectDir, "pom.xml"), "spring-boot") {
		return true
	}
	for _, f := range []string{"build.gradle", "build.gradle.kts"} {
		path := filepath.Join(projectDir, f)
		if FileExists(path) && fileContains(path, "spring-boot") {
			return true
		}
	}
	return false
}

// GenerateClaudeMdScaffold builds the static scaffold content for a project.
// Returns the markdown string (caller handles file I/O and force logic).
func GenerateClaudeMdScaffold(projectDir string) string {
	projectName := filepath.Base(projectDir)
	techStack := "Unknown"
	buildCmd := ""
	testCmd := ""
	lintCmd := ""

	// Detection order: general first, more specific later (last match wins).

	// --- JavaScript/TypeScript (package.json) ---
	pkgPath := filepath.Join(projectDir, "package.json")
	if FileExists(pkgPath) {
		var pkg map[string]json.RawMessage
		if err := ReadJSONFile(pkgPath, &pkg); err == nil {
			techStack = DetectJSTechStack(pkg)
			var scripts map[string]string
			if raw, ok := pkg["scripts"]; ok {
				_ = json.Unmarshal(raw, &scripts)
			}
			if _, ok := scripts["build"]; ok {
				buildCmd = "npm run build"
			}
			if _, ok := scripts["test"]; ok {
				testCmd = "npm test"
			}
			if _, ok := scripts["lint"]; ok {
				lintCmd = "npm run lint"
			}
		}
	}

	// --- Rust (Cargo.toml) ---
	if FileExists(filepath.Join(projectDir, "Cargo.toml")) {
		techStack = "Rust"
		buildCmd = "cargo build"
		testCmd = "cargo test"
		lintCmd = "cargo clippy"
	}

	// --- Python (pyproject.toml or requirements.txt) ---
	if FileExists(filepath.Join(projectDir, "pyproject.toml")) {
		techStack = "Python"
		testCmd = "pytest"
		lintCmd = "ruff check ."
	} else if FileExists(filepath.Join(projectDir, "requirements.txt")) {
		techStack = "Python"
		testCmd = "pytest"
		lintCmd = "ruff check ."
	}

	// --- Go (go.mod) ---
	if FileExists(filepath.Join(projectDir, "go.mod")) {
		techStack = "Go"
		buildCmd = "go build ./..."
		testCmd = "go test ./..."
		lintCmd = "go vet ./..."
	}

	// --- Java Maven (pom.xml) ---
	if FileExists(filepath.Join(projectDir, "pom.xml")) {
		if detectSpringBoot(projectDir) {
			techStack = "Java, Spring Boot, Maven"
		} else {
			techStack = "Java, Maven"
		}
		buildCmd = "mvn package -q"
		testCmd = "mvn test -q"
		lintCmd = "mvn checkstyle:check -q"
	}

	// --- Java/Kotlin Gradle (build.gradle or build.gradle.kts) ---
	if FileExists(filepath.Join(projectDir, "build.gradle")) || FileExists(filepath.Join(projectDir, "build.gradle.kts")) {
		if hasKotlinSources(projectDir) {
			techStack = "Kotlin, Gradle"
			lintCmd = "ktlint"
		} else if detectSpringBoot(projectDir) {
			techStack = "Java, Spring Boot, Gradle"
			lintCmd = "checkstyle"
		} else {
			techStack = "Java, Gradle"
			lintCmd = "checkstyle"
		}
		buildCmd = "./gradlew build"
		testCmd = "./gradlew test"
	}

	// --- Ruby (Gemfile) ---
	if FileExists(filepath.Join(projectDir, "Gemfile")) {
		if FileExists(filepath.Join(projectDir, "config", "routes.rb")) {
			techStack = "Ruby, Rails"
		} else {
			techStack = "Ruby"
		}
		testCmd = "bundle exec rake test"
		lintCmd = "rubocop"
	}

	// --- C# / .NET (*.csproj or *.sln) ---
	if hasGlobMatch(projectDir, "*.csproj") || hasGlobMatch(projectDir, "*.sln") {
		techStack = "C#, .NET"
		buildCmd = "dotnet build"
		testCmd = "dotnet test"
		lintCmd = "dotnet format --verify-no-changes"
	}

	// --- Elixir (mix.exs) ---
	if FileExists(filepath.Join(projectDir, "mix.exs")) {
		if FileExists(filepath.Join(projectDir, "lib", "endpoint.ex")) || FileExists(filepath.Join(projectDir, "lib", "router.ex")) {
			techStack = "Elixir, Phoenix"
		} else {
			techStack = "Elixir"
		}
		buildCmd = "mix compile"
		testCmd = "mix test"
		lintCmd = "mix credo"
	}

	// --- PHP (composer.json) ---
	if FileExists(filepath.Join(projectDir, "composer.json")) {
		if FileExists(filepath.Join(projectDir, "artisan")) {
			techStack = "PHP, Laravel"
		} else {
			techStack = "PHP"
		}
		testCmd = "./vendor/bin/phpunit"
		lintCmd = "phpstan analyse"
	}

	// --- Swift (Package.swift) ---
	if FileExists(filepath.Join(projectDir, "Package.swift")) {
		techStack = "Swift"
		buildCmd = "swift build"
		testCmd = "swift test"
		lintCmd = "swiftlint"
	}

	// --- Scala (build.sbt) ---
	if FileExists(filepath.Join(projectDir, "build.sbt")) {
		techStack = "Scala"
		buildCmd = "sbt compile"
		testCmd = "sbt test"
		lintCmd = "scalafmt --check"
	}

	// --- C++ with CMake ---
	if FileExists(filepath.Join(projectDir, "CMakeLists.txt")) {
		techStack = "C++, CMake"
		buildCmd = "cmake --build build"
		testCmd = "ctest --test-dir build"
		lintCmd = "clang-tidy"
	}

	// --- Bazel (MODULE.bazel or WORKSPACE) ---
	if FileExists(filepath.Join(projectDir, "MODULE.bazel")) || FileExists(filepath.Join(projectDir, "WORKSPACE")) {
		if hasCppSources(projectDir) {
			techStack = "C++, Bazel"
		} else {
			techStack = "Bazel"
		}
		buildCmd = "bazel build //..."
		testCmd = "bazel test //..."
	}

	// --- C++ with Makefile (only if C++ source files exist) ---
	if FileExists(filepath.Join(projectDir, "Makefile")) && hasCppSources(projectDir) {
		techStack = "C++, Make"
		buildCmd = "make"
		testCmd = "make test"
		lintCmd = "clang-tidy"
	}

	var sb strings.Builder
	sb.WriteString("# Project Instructions\n\n")
	sb.WriteString("## Project\n")
	fmt.Fprintf(&sb, "Name: %s\n", projectName)
	fmt.Fprintf(&sb, "Tech Stack: %s\n", techStack)
	if buildCmd != "" {
		fmt.Fprintf(&sb, "Build: `%s`\n", buildCmd)
	}
	if testCmd != "" {
		fmt.Fprintf(&sb, "Test: `%s`\n", testCmd)
	}
	if lintCmd != "" {
		fmt.Fprintf(&sb, "Lint: `%s`\n", lintCmd)
	}
	sb.WriteString(`
## Conventions
<!-- Add your team's coding conventions here -->

## Key Directories
<!-- Map your project's important directories -->
<!-- Example:
- src/          - Application source code
- tests/        - Test files
- docs/         - Documentation
-->

## Important Notes
<!-- Add project-specific notes for Claude -->
`)

	return sb.String()
}

// BuildEnrichmentPrompt constructs the LLM prompt for AI enrichment.
func BuildEnrichmentPrompt(projectDir, claudeMdPath string) string {
	return fmt.Sprintf(`You are analyzing a software project to generate a CLAUDE.md file with real project context.

The project is located at: %s
There is an existing scaffold at: %s

Your task:
1. Read the existing scaffold at the path above
2. Explore the project: README, dependency files, directory layout, config files, and source files
3. Output ONLY raw markdown (no code fences, no explanations, no preamble) following this exact structure:

# Project Instructions

## Project
Name: <project name>
Purpose: <one-line description from README or package metadata>
Tech Stack: <detected languages/frameworks>
Build: `+"`<build command>`"+`
Test: `+"`<test command>`"+`
Lint: `+"`<lint command if found>`"+`

## Key Directories
- <dir>/ - <description>
(list actual directories found in the project)

## Conventions
- <convention discovered from code>
(e.g., naming patterns, file organization, import style, error handling patterns)

## Important Files
- <file path> - <why it matters>
(list 5-10 files a new developer should read first)

## Important Notes
- <project-specific gotcha or important detail>

Rules:
- Only include information you can verify from the project files
- Do not hallucinate or guess — if unsure, omit the section content
- Keep the total output under 150 lines
- Output raw markdown only — no wrapping code fences, no commentary`, projectDir, claudeMdPath)
}

// EnrichClaudeMd runs claude opus to analyze the project and overwrite CLAUDE.md.
func EnrichClaudeMd(projectDir, claudeDir string) error {
	if !Exists("claude") {
		return fmt.Errorf("claude CLI not found. Install with `claude-workspace setup`")
	}

	claudeMdPath := filepath.Join(claudeDir, "CLAUDE.md")
	prompt := BuildEnrichmentPrompt(projectDir, claudeMdPath)

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	spinner := StartSpinner(os.Stderr, "Analyzing project with claude opus (up to 180s)...")
	stdout, stderr, err := RunDirWithStdinCapture(ctx, projectDir, prompt, []string{"CLAUDECODE"}, "claude", "-p",
		"--strict-mcp-config", "--mcp-config", `{"mcpServers":{}}`,
		"--output-format", "text",
		"--model", "opus")
	spinner.Stop()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("enrichment timed out after 180s")
		}
		detail := ""
		if stderr != "" {
			detail = fmt.Sprintf(": %s", stderr)
		}
		return fmt.Errorf("claude exited with error (check API key with `claude-workspace setup`)%s", detail)
	}

	// Find the start of markdown content (skip any preamble lines)
	idx := strings.Index(stdout, "#")
	if idx < 0 {
		if stderr != "" {
			return fmt.Errorf("enrichment produced no markdown output (stderr: %s)", stderr)
		}
		return fmt.Errorf("enrichment produced no markdown output")
	}
	content := stdout[idx:]

	if err := os.WriteFile(claudeMdPath, []byte(content+"\n"), 0644); err != nil {
		return fmt.Errorf("writing enriched CLAUDE.md: %w", err)
	}

	PrintSuccess(os.Stdout, "Enriched .claude/CLAUDE.md with project context")
	return nil
}
