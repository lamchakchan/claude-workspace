package platform

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makePkg(deps map[string]string, devDeps map[string]string) map[string]json.RawMessage {
	pkg := make(map[string]json.RawMessage)
	if deps != nil {
		b, _ := json.Marshal(deps)
		pkg["dependencies"] = b
	}
	if devDeps != nil {
		b, _ := json.Marshal(devDeps)
		pkg["devDependencies"] = b
	}
	return pkg
}

func TestDetectJSTechStack(t *testing.T) {
	tests := []struct {
		name    string
		deps    map[string]string
		devDeps map[string]string
		want    string
	}{
		{
			name: "empty deps returns Node.js",
			deps: map[string]string{},
			want: "Node.js",
		},
		{
			name: "nil deps returns Node.js",
			want: "Node.js",
		},
		{
			name: "react project",
			deps: map[string]string{"react": "^18.0.0"},
			want: "React",
		},
		{
			name: "next.js project (prefers Next.js over React)",
			deps: map[string]string{"next": "^14.0.0", "react": "^18.0.0"},
			want: "Next.js",
		},
		{
			name: "vue project",
			deps: map[string]string{"vue": "^3.0.0"},
			want: "Vue",
		},
		{
			name: "angular project",
			deps: map[string]string{"@angular/core": "^17.0.0"},
			want: "Angular",
		},
		{
			name: "svelte project",
			deps: map[string]string{"svelte": "^4.0.0"},
			want: "Svelte",
		},
		{
			name:    "react with typescript",
			deps:    map[string]string{"react": "^18.0.0"},
			devDeps: map[string]string{"typescript": "^5.0.0"},
			want:    "React, TypeScript",
		},
		{
			name: "express backend",
			deps: map[string]string{"express": "^4.0.0"},
			want: "Express",
		},
		{
			name: "fastify backend",
			deps: map[string]string{"fastify": "^4.0.0"},
			want: "Fastify",
		},
		{
			name: "hono backend",
			deps: map[string]string{"hono": "^4.0.0"},
			want: "Hono",
		},
		{
			name: "next.js + express + typescript + prisma",
			deps: map[string]string{
				"next":           "^14.0.0",
				"react":          "^18.0.0",
				"express":        "^4.0.0",
				"@prisma/client": "^5.0.0",
			},
			devDeps: map[string]string{"typescript": "^5.0.0"},
			want:    "Next.js, Express, TypeScript, Prisma",
		},
		{
			name:    "prisma via prisma package",
			devDeps: map[string]string{"prisma": "^5.0.0"},
			want:    "Prisma",
		},
		{
			name: "drizzle ORM",
			deps: map[string]string{"drizzle": "^0.30.0"},
			want: "Drizzle",
		},
		{
			name:    "typescript only in devDeps",
			devDeps: map[string]string{"typescript": "^5.0.0"},
			want:    "TypeScript",
		},
		{
			name: "multiple backend frameworks",
			deps: map[string]string{"express": "^4.0.0", "fastify": "^4.0.0"},
			want: "Express, Fastify",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := makePkg(tt.deps, tt.devDeps)
			got := DetectJSTechStack(pkg)
			if got != tt.want {
				t.Errorf("DetectJSTechStack() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildEnrichmentPrompt(t *testing.T) {
	prompt := BuildEnrichmentPrompt("/tmp/my-project", "/tmp/my-project/.claude/CLAUDE.md")

	// Verify it includes the project path and CLAUDE.md path
	if !strings.Contains(prompt, "/tmp/my-project") {
		t.Error("prompt should contain project directory path")
	}
	if !strings.Contains(prompt, "/tmp/my-project/.claude/CLAUDE.md") {
		t.Error("prompt should contain CLAUDE.md path")
	}

	// Verify it includes the expected section headers
	for _, section := range []string{
		"# Project Instructions",
		"## Project",
		"## Key Directories",
		"## Conventions",
		"## Important Files",
		"## Important Notes",
	} {
		if !strings.Contains(prompt, section) {
			t.Errorf("prompt should contain section %q", section)
		}
	}

	// Verify it includes key instructions
	if !strings.Contains(prompt, "under 150 lines") {
		t.Error("prompt should mention the 150 line limit")
	}
	if !strings.Contains(prompt, "no code fences") {
		t.Error("prompt should instruct no code fences")
	}
}

func TestEnrichClaudeMd_MissingCLI(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	_ = os.MkdirAll(claudeDir, 0755)
	_ = os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte("# scaffold"), 0644)

	// Save and clear PATH so claude won't be found
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", origPath)

	err := EnrichClaudeMd(dir, claudeDir)
	if err == nil {
		t.Fatal("EnrichClaudeMd() expected error when claude CLI missing")
	}
	if !strings.Contains(err.Error(), "claude CLI not found") {
		t.Errorf("error = %q, want to contain 'claude CLI not found'", err.Error())
	}

	// Verify scaffold is preserved
	data, _ := os.ReadFile(filepath.Join(claudeDir, "CLAUDE.md"))
	if string(data) != "# scaffold" {
		t.Errorf("scaffold was modified: got %q", string(data))
	}
}

// testEnrichClaudeMdFailure is a helper for testing EnrichClaudeMd error scenarios.
// It creates a fake claude script, runs enrichment, and verifies the expected error
// and that the scaffold file is preserved.
func testEnrichClaudeMdFailure(t *testing.T, scriptBody, wantErrContains string) {
	t.Helper()
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	_ = os.MkdirAll(claudeDir, 0755)
	_ = os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte("# scaffold"), 0644)

	binDir := filepath.Join(dir, "bin")
	_ = os.MkdirAll(binDir, 0755)
	fakeScript := filepath.Join(binDir, "claude")
	_ = os.WriteFile(fakeScript, []byte(scriptBody), 0755)

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", binDir)
	defer os.Setenv("PATH", origPath)

	err := EnrichClaudeMd(dir, claudeDir)
	if err == nil {
		t.Fatal("EnrichClaudeMd() expected error")
	}
	if !strings.Contains(err.Error(), wantErrContains) {
		t.Errorf("error = %q, want to contain %q", err.Error(), wantErrContains)
	}

	// Verify scaffold is preserved
	data, _ := os.ReadFile(filepath.Join(claudeDir, "CLAUDE.md"))
	if string(data) != "# scaffold" {
		t.Errorf("scaffold was modified: got %q", string(data))
	}
}

func TestEnrichClaudeMd_ScriptReturnsInvalidOutput(t *testing.T) {
	testEnrichClaudeMdFailure(t, "#!/bin/sh\necho 'not markdown'\n", "enrichment produced no markdown output")
}

func TestEnrichClaudeMd_ScriptReturnsEmptyOutput(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	_ = os.MkdirAll(claudeDir, 0755)
	_ = os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte("# scaffold"), 0644)

	// Create a fake "claude" script that outputs nothing
	binDir := filepath.Join(dir, "bin")
	_ = os.MkdirAll(binDir, 0755)
	fakeScript := filepath.Join(binDir, "claude")
	_ = os.WriteFile(fakeScript, []byte("#!/bin/sh\n"), 0755)

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", binDir)
	defer os.Setenv("PATH", origPath)

	err := EnrichClaudeMd(dir, claudeDir)
	if err == nil {
		t.Fatal("EnrichClaudeMd() expected error for empty output")
	}
	if !strings.Contains(err.Error(), "enrichment produced no markdown output") {
		t.Errorf("error = %q, want to contain 'enrichment produced no markdown output'", err.Error())
	}
}

func TestEnrichClaudeMd_ScriptReturnsValidOutput(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	_ = os.MkdirAll(claudeDir, 0755)
	_ = os.WriteFile(filepath.Join(claudeDir, "CLAUDE.md"), []byte("# scaffold"), 0644)

	// Create a fake "claude" script that outputs valid enriched markdown
	binDir := filepath.Join(dir, "bin")
	_ = os.MkdirAll(binDir, 0755)
	enrichedContent := "# Project Instructions\n\n## Project\nName: test-project"
	fakeScript := filepath.Join(binDir, "claude")
	_ = os.WriteFile(fakeScript, []byte("#!/bin/sh\nprintf '"+enrichedContent+"'\n"), 0755)

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", binDir)
	defer os.Setenv("PATH", origPath)

	err := EnrichClaudeMd(dir, claudeDir)
	if err != nil {
		t.Fatalf("EnrichClaudeMd() unexpected error: %v", err)
	}

	// Verify file was overwritten with enriched content
	data, _ := os.ReadFile(filepath.Join(claudeDir, "CLAUDE.md"))
	if !strings.HasPrefix(string(data), "# Project Instructions") {
		t.Errorf("expected enriched content, got %q", string(data))
	}
}

func TestEnrichClaudeMd_ScriptFailsNonZero(t *testing.T) {
	testEnrichClaudeMdFailure(t, "#!/bin/sh\nexit 1\n", "claude exited with error")
}

func TestGenerateClaudeMdScaffold(t *testing.T) {
	dir := t.TempDir()

	// Test with a Go project
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)

	scaffold := GenerateClaudeMdScaffold(dir)

	if !strings.Contains(scaffold, "# Project Instructions") {
		t.Error("scaffold should contain header")
	}
	if !strings.Contains(scaffold, "Tech Stack: Go") {
		t.Error("scaffold should detect Go tech stack")
	}
	if !strings.Contains(scaffold, "Build: `go build ./...`") {
		t.Error("scaffold should include Go build command")
	}
	if !strings.Contains(scaffold, "Test: `go test ./...`") {
		t.Error("scaffold should include Go test command")
	}
	if !strings.Contains(scaffold, "Lint: `go vet ./...`") {
		t.Error("scaffold should include Go lint command")
	}
}

func TestGenerateClaudeMdScaffold_AllLanguages(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(dir string)
		wantStack string
		wantBuild string
		wantTest  string
		wantLint  string
	}{
		{
			name: "Rust project",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]"), 0644)
			},
			wantStack: "Rust",
			wantBuild: "cargo build",
			wantTest:  "cargo test",
			wantLint:  "cargo clippy",
		},
		{
			name: "Python pyproject.toml",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]"), 0644)
			},
			wantStack: "Python",
			wantTest:  "pytest",
			wantLint:  "ruff check .",
		},
		{
			name: "Python requirements.txt",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("flask==2.0"), 0644)
			},
			wantStack: "Python",
			wantTest:  "pytest",
			wantLint:  "ruff check .",
		},
		{
			name: "Java Maven",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project><groupId>com.example</groupId></project>"), 0644)
			},
			wantStack: "Java, Maven",
			wantBuild: "mvn package -q",
			wantTest:  "mvn test -q",
			wantLint:  "mvn checkstyle:check -q",
		},
		{
			name: "Java Maven with Spring Boot",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project><parent><artifactId>spring-boot-starter-parent</artifactId></parent></project>"), 0644)
			},
			wantStack: "Java, Spring Boot, Maven",
			wantBuild: "mvn package -q",
			wantTest:  "mvn test -q",
			wantLint:  "mvn checkstyle:check -q",
		},
		{
			name: "Java Gradle",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "build.gradle"), []byte("apply plugin: 'java'"), 0644)
			},
			wantStack: "Java, Gradle",
			wantBuild: "./gradlew build",
			wantTest:  "./gradlew test",
			wantLint:  "checkstyle",
		},
		{
			name: "Java Gradle with Spring Boot",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "build.gradle"), []byte("plugins { id 'org.springframework.boot' }\ndependencies { implementation 'spring-boot' }"), 0644)
			},
			wantStack: "Java, Spring Boot, Gradle",
			wantBuild: "./gradlew build",
			wantTest:  "./gradlew test",
			wantLint:  "checkstyle",
		},
		{
			name: "Kotlin Gradle",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "build.gradle.kts"), []byte("plugins { kotlin(\"jvm\") }"), 0644)
				_ = os.MkdirAll(filepath.Join(dir, "src"), 0755)
				_ = os.WriteFile(filepath.Join(dir, "src", "Main.kt"), []byte("fun main() {}"), 0644)
			},
			wantStack: "Kotlin, Gradle",
			wantBuild: "./gradlew build",
			wantTest:  "./gradlew test",
			wantLint:  "ktlint",
		},
		{
			name: "Ruby",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "Gemfile"), []byte("source 'https://rubygems.org'"), 0644)
			},
			wantStack: "Ruby",
			wantTest:  "bundle exec rake test",
			wantLint:  "rubocop",
		},
		{
			name: "Ruby on Rails",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "Gemfile"), []byte("gem 'rails'"), 0644)
				_ = os.MkdirAll(filepath.Join(dir, "config"), 0755)
				_ = os.WriteFile(filepath.Join(dir, "config", "routes.rb"), []byte("Rails.application.routes.draw do\nend"), 0644)
			},
			wantStack: "Ruby, Rails",
			wantTest:  "bundle exec rake test",
			wantLint:  "rubocop",
		},
		{
			name: "C# .NET via csproj",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "MyApp.csproj"), []byte("<Project></Project>"), 0644)
			},
			wantStack: "C#, .NET",
			wantBuild: "dotnet build",
			wantTest:  "dotnet test",
			wantLint:  "dotnet format --verify-no-changes",
		},
		{
			name: "C# .NET via sln",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "MyApp.sln"), []byte("Microsoft Visual Studio Solution"), 0644)
			},
			wantStack: "C#, .NET",
			wantBuild: "dotnet build",
			wantTest:  "dotnet test",
			wantLint:  "dotnet format --verify-no-changes",
		},
		{
			name: "Elixir",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "mix.exs"), []byte("defmodule MyApp.MixProject do\nend"), 0644)
			},
			wantStack: "Elixir",
			wantBuild: "mix compile",
			wantTest:  "mix test",
			wantLint:  "mix credo",
		},
		{
			name: "Elixir Phoenix",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "mix.exs"), []byte("defmodule MyApp.MixProject do\nend"), 0644)
				_ = os.MkdirAll(filepath.Join(dir, "lib"), 0755)
				_ = os.WriteFile(filepath.Join(dir, "lib", "endpoint.ex"), []byte("defmodule Endpoint do\nend"), 0644)
			},
			wantStack: "Elixir, Phoenix",
			wantBuild: "mix compile",
			wantTest:  "mix test",
			wantLint:  "mix credo",
		},
		{
			name: "PHP",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "composer.json"), []byte("{}"), 0644)
			},
			wantStack: "PHP",
			wantTest:  "./vendor/bin/phpunit",
			wantLint:  "phpstan analyse",
		},
		{
			name: "PHP Laravel",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "composer.json"), []byte("{}"), 0644)
				_ = os.WriteFile(filepath.Join(dir, "artisan"), []byte("#!/usr/bin/env php"), 0644)
			},
			wantStack: "PHP, Laravel",
			wantTest:  "./vendor/bin/phpunit",
			wantLint:  "phpstan analyse",
		},
		{
			name: "Swift",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "Package.swift"), []byte("// swift-tools-version:5.9"), 0644)
			},
			wantStack: "Swift",
			wantBuild: "swift build",
			wantTest:  "swift test",
			wantLint:  "swiftlint",
		},
		{
			name: "Scala",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "build.sbt"), []byte("name := \"myapp\""), 0644)
			},
			wantStack: "Scala",
			wantBuild: "sbt compile",
			wantTest:  "sbt test",
			wantLint:  "scalafmt --check",
		},
		{
			name: "C++ CMake",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.14)"), 0644)
			},
			wantStack: "C++, CMake",
			wantBuild: "cmake --build build",
			wantTest:  "ctest --test-dir build",
			wantLint:  "clang-tidy",
		},
		{
			name: "C++ Bazel",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "MODULE.bazel"), []byte("module()"), 0644)
				_ = os.WriteFile(filepath.Join(dir, "main.cpp"), []byte("int main() {}"), 0644)
			},
			wantStack: "C++, Bazel",
			wantBuild: "bazel build //...",
			wantTest:  "bazel test //...",
		},
		{
			name: "Bazel without C++",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "MODULE.bazel"), []byte("module()"), 0644)
			},
			wantStack: "Bazel",
			wantBuild: "bazel build //...",
			wantTest:  "bazel test //...",
		},
		{
			name: "Bazel via WORKSPACE",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "WORKSPACE"), []byte(""), 0644)
			},
			wantStack: "Bazel",
			wantBuild: "bazel build //...",
			wantTest:  "bazel test //...",
		},
		{
			name: "C++ Make",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "Makefile"), []byte("all:\n\tg++ main.cpp"), 0644)
				_ = os.WriteFile(filepath.Join(dir, "main.cpp"), []byte("int main() {}"), 0644)
			},
			wantStack: "C++, Make",
			wantBuild: "make",
			wantTest:  "make test",
			wantLint:  "clang-tidy",
		},
		{
			name: "Makefile without C++ sources",
			setup: func(dir string) {
				_ = os.WriteFile(filepath.Join(dir, "Makefile"), []byte("all:\n\techo hi"), 0644)
			},
			wantStack: "Unknown",
		},
		{
			name: "Unknown project",
			setup: func(_ string) {
				// empty directory
			},
			wantStack: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)

			scaffold := GenerateClaudeMdScaffold(dir)

			if !strings.Contains(scaffold, "Tech Stack: "+tt.wantStack) {
				t.Errorf("want Tech Stack: %s, got scaffold:\n%s", tt.wantStack, scaffold)
			}

			if tt.wantBuild != "" {
				if !strings.Contains(scaffold, "Build: `"+tt.wantBuild+"`") {
					t.Errorf("want Build: `%s` in scaffold", tt.wantBuild)
				}
			}

			if tt.wantTest != "" {
				if !strings.Contains(scaffold, "Test: `"+tt.wantTest+"`") {
					t.Errorf("want Test: `%s` in scaffold", tt.wantTest)
				}
			}

			if tt.wantLint != "" {
				if !strings.Contains(scaffold, "Lint: `"+tt.wantLint+"`") {
					t.Errorf("want Lint: `%s` in scaffold", tt.wantLint)
				}
			}
		})
	}
}
