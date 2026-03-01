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

// projectConfig holds detected build/test/lint commands for a project.
type projectConfig struct {
	techStack string
	buildCmd  string
	testCmd   string
	lintCmd   string
}

// projectDetectors is the ordered list of language detectors.
// Detection order: general first, more specific later (last match wins).
var projectDetectors = []func(string, *projectConfig){
	detectJSProject,
	detectRustProject,
	detectPythonProject,
	detectGoProject,
	detectMavenProject,
	detectGradleProject,
	detectRubyProject,
	detectDotNetProject,
	detectElixirProject,
	detectPHPProject,
	detectSwiftProject,
	detectScalaProject,
	detectCMakeProject,
	detectBazelProject,
	detectCppMakeProject,
}

func detectJSProject(dir string, cfg *projectConfig) {
	pkgPath := filepath.Join(dir, "package.json")
	if !FileExists(pkgPath) {
		return
	}
	var pkg map[string]json.RawMessage
	if err := ReadJSONFile(pkgPath, &pkg); err != nil {
		return
	}
	cfg.techStack = DetectJSTechStack(pkg)
	var scripts map[string]string
	if raw, ok := pkg["scripts"]; ok {
		_ = json.Unmarshal(raw, &scripts)
	}
	if _, ok := scripts["build"]; ok {
		cfg.buildCmd = "npm run build"
	}
	if _, ok := scripts["test"]; ok {
		cfg.testCmd = "npm test"
	}
	if _, ok := scripts["lint"]; ok {
		cfg.lintCmd = "npm run lint"
	}
}

func detectRustProject(dir string, cfg *projectConfig) {
	if !FileExists(filepath.Join(dir, "Cargo.toml")) {
		return
	}
	cfg.techStack = "Rust"
	cfg.buildCmd = "cargo build"
	cfg.testCmd = "cargo test"
	cfg.lintCmd = "cargo clippy"
}

func detectPythonProject(dir string, cfg *projectConfig) {
	if !FileExists(filepath.Join(dir, "pyproject.toml")) && !FileExists(filepath.Join(dir, "requirements.txt")) {
		return
	}
	cfg.techStack = "Python"
	cfg.testCmd = "pytest"
	cfg.lintCmd = "ruff check ."
}

func detectGoProject(dir string, cfg *projectConfig) {
	if !FileExists(filepath.Join(dir, "go.mod")) {
		return
	}
	cfg.techStack = "Go"
	cfg.buildCmd = "go build ./..."
	cfg.testCmd = "go test ./..."
	cfg.lintCmd = "go vet ./..."
}

func detectMavenProject(dir string, cfg *projectConfig) {
	if !FileExists(filepath.Join(dir, "pom.xml")) {
		return
	}
	if detectSpringBoot(dir) {
		cfg.techStack = "Java, Spring Boot, Maven"
	} else {
		cfg.techStack = "Java, Maven"
	}
	cfg.buildCmd = "mvn package -q"
	cfg.testCmd = "mvn test -q"
	cfg.lintCmd = "mvn checkstyle:check -q"
}

func detectGradleProject(dir string, cfg *projectConfig) {
	if !FileExists(filepath.Join(dir, "build.gradle")) && !FileExists(filepath.Join(dir, "build.gradle.kts")) {
		return
	}
	switch {
	case hasKotlinSources(dir):
		cfg.techStack = "Kotlin, Gradle"
		cfg.lintCmd = "ktlint"
	case detectSpringBoot(dir):
		cfg.techStack = "Java, Spring Boot, Gradle"
		cfg.lintCmd = "checkstyle"
	default:
		cfg.techStack = "Java, Gradle"
		cfg.lintCmd = "checkstyle"
	}
	cfg.buildCmd = "./gradlew build"
	cfg.testCmd = "./gradlew test"
}

func detectRubyProject(dir string, cfg *projectConfig) {
	if !FileExists(filepath.Join(dir, "Gemfile")) {
		return
	}
	if FileExists(filepath.Join(dir, "config", "routes.rb")) {
		cfg.techStack = "Ruby, Rails"
	} else {
		cfg.techStack = "Ruby"
	}
	cfg.testCmd = "bundle exec rake test"
	cfg.lintCmd = "rubocop"
}

func detectDotNetProject(dir string, cfg *projectConfig) {
	if !hasGlobMatch(dir, "*.csproj") && !hasGlobMatch(dir, "*.sln") {
		return
	}
	cfg.techStack = "C#, .NET"
	cfg.buildCmd = "dotnet build"
	cfg.testCmd = "dotnet test"
	cfg.lintCmd = "dotnet format --verify-no-changes"
}

func detectElixirProject(dir string, cfg *projectConfig) {
	if !FileExists(filepath.Join(dir, "mix.exs")) {
		return
	}
	if FileExists(filepath.Join(dir, "lib", "endpoint.ex")) || FileExists(filepath.Join(dir, "lib", "router.ex")) {
		cfg.techStack = "Elixir, Phoenix"
	} else {
		cfg.techStack = "Elixir"
	}
	cfg.buildCmd = "mix compile"
	cfg.testCmd = "mix test"
	cfg.lintCmd = "mix credo"
}

func detectPHPProject(dir string, cfg *projectConfig) {
	if !FileExists(filepath.Join(dir, "composer.json")) {
		return
	}
	if FileExists(filepath.Join(dir, "artisan")) {
		cfg.techStack = "PHP, Laravel"
	} else {
		cfg.techStack = "PHP"
	}
	cfg.testCmd = "./vendor/bin/phpunit"
	cfg.lintCmd = "phpstan analyse"
}

func detectSwiftProject(dir string, cfg *projectConfig) {
	if !FileExists(filepath.Join(dir, "Package.swift")) {
		return
	}
	cfg.techStack = "Swift"
	cfg.buildCmd = "swift build"
	cfg.testCmd = "swift test"
	cfg.lintCmd = "swiftlint"
}

func detectScalaProject(dir string, cfg *projectConfig) {
	if !FileExists(filepath.Join(dir, "build.sbt")) {
		return
	}
	cfg.techStack = "Scala"
	cfg.buildCmd = "sbt compile"
	cfg.testCmd = "sbt test"
	cfg.lintCmd = "scalafmt --check"
}

func detectCMakeProject(dir string, cfg *projectConfig) {
	if !FileExists(filepath.Join(dir, "CMakeLists.txt")) {
		return
	}
	cfg.techStack = "C++, CMake"
	cfg.buildCmd = "cmake --build build"
	cfg.testCmd = "ctest --test-dir build"
	cfg.lintCmd = "clang-tidy"
}

func detectBazelProject(dir string, cfg *projectConfig) {
	if !FileExists(filepath.Join(dir, "MODULE.bazel")) && !FileExists(filepath.Join(dir, "WORKSPACE")) {
		return
	}
	if hasCppSources(dir) {
		cfg.techStack = "C++, Bazel"
	} else {
		cfg.techStack = "Bazel"
	}
	cfg.buildCmd = "bazel build //..."
	cfg.testCmd = "bazel test //..."
}

func detectCppMakeProject(dir string, cfg *projectConfig) {
	if !FileExists(filepath.Join(dir, "Makefile")) || !hasCppSources(dir) {
		return
	}
	cfg.techStack = "C++, Make"
	cfg.buildCmd = "make"
	cfg.testCmd = "make test"
	cfg.lintCmd = "clang-tidy"
}

// GenerateClaudeMdScaffold builds the static scaffold content for a project.
// Returns the markdown string (caller handles file I/O and force logic).
func GenerateClaudeMdScaffold(projectDir string) string {
	projectName := filepath.Base(projectDir)

	cfg := projectConfig{techStack: "Unknown"}
	for _, detect := range projectDetectors {
		detect(projectDir, &cfg)
	}

	var sb strings.Builder
	sb.WriteString("# Project Instructions\n\n")
	sb.WriteString("## Project\n")
	fmt.Fprintf(&sb, "Name: %s\n", projectName)
	fmt.Fprintf(&sb, "Tech Stack: %s\n", cfg.techStack)
	if cfg.buildCmd != "" {
		fmt.Fprintf(&sb, "Build: `%s`\n", cfg.buildCmd)
	}
	if cfg.testCmd != "" {
		fmt.Fprintf(&sb, "Test: `%s`\n", cfg.testCmd)
	}
	if cfg.lintCmd != "" {
		fmt.Fprintf(&sb, "Lint: `%s`\n", cfg.lintCmd)
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
