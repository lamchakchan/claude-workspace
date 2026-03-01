#!/bin/bash
set -euo pipefail

# TaskCompleted hook: runs project tests before allowing task completion.
# Exit 0 = allow, Exit 2 = block with feedback.
# Fails open: if no test framework detected or input is missing, allows completion.

INPUT=$(cat)
TASK_SUBJECT=$(echo "$INPUT" | jq -r '.task_subject // empty' 2>/dev/null || true)

# If we can't read input, fail open
if [ -z "$TASK_SUBJECT" ]; then
  exit 0
fi

PROJECT_DIR="${CLAUDE_PROJECT_DIR:-.}"

# Detect project type and run tests (most-specific first)
if [ -f "$PROJECT_DIR/MODULE.bazel" ] || [ -f "$PROJECT_DIR/WORKSPACE" ] || [ -f "$PROJECT_DIR/BUILD.bazel" ]; then
  echo "Running Bazel tests..." >&2
  if ! (cd "$PROJECT_DIR" && bazel test //... 2>&1); then
    echo "BLOCKED: Bazel tests failed. Fix test failures before completing task: $TASK_SUBJECT" >&2
    exit 2
  fi
elif [ -f "$PROJECT_DIR/CMakeLists.txt" ]; then
  echo "Running CMake tests..." >&2
  if ! (cd "$PROJECT_DIR" && cmake --build build 2>&1 && ctest --test-dir build 2>&1); then
    echo "BLOCKED: CMake tests failed. Fix test failures before completing task: $TASK_SUBJECT" >&2
    exit 2
  fi
elif [ -f "$PROJECT_DIR/mix.exs" ]; then
  echo "Running Elixir tests..." >&2
  if ! (cd "$PROJECT_DIR" && mix test 2>&1); then
    echo "BLOCKED: Elixir tests failed. Fix test failures before completing task: $TASK_SUBJECT" >&2
    exit 2
  fi
elif [ -f "$PROJECT_DIR/Package.swift" ]; then
  echo "Running Swift tests..." >&2
  if ! (cd "$PROJECT_DIR" && swift test 2>&1); then
    echo "BLOCKED: Swift tests failed. Fix test failures before completing task: $TASK_SUBJECT" >&2
    exit 2
  fi
elif [ -f "$PROJECT_DIR/build.sbt" ]; then
  echo "Running Scala tests..." >&2
  if ! (cd "$PROJECT_DIR" && sbt test 2>&1); then
    echo "BLOCKED: Scala tests failed. Fix test failures before completing task: $TASK_SUBJECT" >&2
    exit 2
  fi
elif [ -f "$PROJECT_DIR/build.gradle" ] || [ -f "$PROJECT_DIR/build.gradle.kts" ]; then
  echo "Running Gradle tests..." >&2
  if ! (cd "$PROJECT_DIR" && ./gradlew test 2>&1); then
    echo "BLOCKED: Gradle tests failed. Fix test failures before completing task: $TASK_SUBJECT" >&2
    exit 2
  fi
elif [ -f "$PROJECT_DIR/pom.xml" ]; then
  echo "Running Maven tests..." >&2
  if ! (cd "$PROJECT_DIR" && mvn test -q 2>&1); then
    echo "BLOCKED: Maven tests failed. Fix test failures before completing task: $TASK_SUBJECT" >&2
    exit 2
  fi
elif [ -f "$PROJECT_DIR/Gemfile" ]; then
  echo "Running Ruby tests..." >&2
  if [ -d "$PROJECT_DIR/spec" ]; then
    if ! (cd "$PROJECT_DIR" && bundle exec rspec 2>&1); then
      echo "BLOCKED: Ruby tests failed. Fix test failures before completing task: $TASK_SUBJECT" >&2
      exit 2
    fi
  else
    if ! (cd "$PROJECT_DIR" && bundle exec rake test 2>&1); then
      echo "BLOCKED: Ruby tests failed. Fix test failures before completing task: $TASK_SUBJECT" >&2
      exit 2
    fi
  fi
elif ls "$PROJECT_DIR"/*.csproj "$PROJECT_DIR"/*.sln 2>/dev/null | head -1 | grep -q .; then
  echo "Running .NET tests..." >&2
  if ! (cd "$PROJECT_DIR" && dotnet test 2>&1); then
    echo "BLOCKED: .NET tests failed. Fix test failures before completing task: $TASK_SUBJECT" >&2
    exit 2
  fi
elif [ -f "$PROJECT_DIR/composer.json" ] && [ -f "$PROJECT_DIR/vendor/bin/phpunit" ]; then
  echo "Running PHP tests..." >&2
  if ! (cd "$PROJECT_DIR" && ./vendor/bin/phpunit 2>&1); then
    echo "BLOCKED: PHP tests failed. Fix test failures before completing task: $TASK_SUBJECT" >&2
    exit 2
  fi
elif [ -f "$PROJECT_DIR/requirements.txt" ] && [ ! -f "$PROJECT_DIR/pyproject.toml" ]; then
  if command -v pytest &>/dev/null; then
    echo "Running Python tests..." >&2
    if ! (cd "$PROJECT_DIR" && pytest 2>&1); then
      echo "BLOCKED: Python tests failed. Fix test failures before completing task: $TASK_SUBJECT" >&2
      exit 2
    fi
  fi
elif [ -f "$PROJECT_DIR/go.mod" ]; then
  echo "Running Go tests..." >&2
  if ! (cd "$PROJECT_DIR" && go test ./... 2>&1); then
    echo "BLOCKED: Go tests failed. Fix test failures before completing task: $TASK_SUBJECT" >&2
    exit 2
  fi
  if ! (cd "$PROJECT_DIR" && go vet ./... 2>&1); then
    echo "BLOCKED: go vet found issues. Fix vet warnings before completing task: $TASK_SUBJECT" >&2
    exit 2
  fi
elif [ -f "$PROJECT_DIR/package.json" ]; then
  if grep -q '"test"' "$PROJECT_DIR/package.json" 2>/dev/null; then
    echo "Running npm tests..." >&2
    if ! (cd "$PROJECT_DIR" && npm test 2>&1); then
      echo "BLOCKED: npm tests failed. Fix test failures before completing task: $TASK_SUBJECT" >&2
      exit 2
    fi
  fi
elif [ -f "$PROJECT_DIR/Cargo.toml" ]; then
  echo "Running Cargo tests..." >&2
  if ! (cd "$PROJECT_DIR" && cargo test 2>&1); then
    echo "BLOCKED: Cargo tests failed. Fix test failures before completing task: $TASK_SUBJECT" >&2
    exit 2
  fi
elif [ -f "$PROJECT_DIR/Makefile" ] && ls "$PROJECT_DIR"/*.cpp "$PROJECT_DIR"/*.cc "$PROJECT_DIR"/*.cxx "$PROJECT_DIR"/src/*.cpp "$PROJECT_DIR"/src/*.cc 2>/dev/null | head -1 | grep -q .; then
  echo "Running C++ tests..." >&2
  if ! (cd "$PROJECT_DIR" && make test 2>&1); then
    echo "BLOCKED: C++ tests failed. Fix test failures before completing task: $TASK_SUBJECT" >&2
    exit 2
  fi
fi

# No test framework detected or tests passed
exit 0
