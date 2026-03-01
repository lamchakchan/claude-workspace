#!/bin/bash
set -euo pipefail

# Expand PATH for version managers (asdf, nvm, homebrew) since hooks run in a non-login shell
export PATH="$HOME/.asdf/shims:$HOME/.local/bin:/opt/homebrew/bin:/usr/local/bin:$PATH"

# Auto-formats files after write/edit operations
INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

if [ -z "$FILE_PATH" ] || [ ! -f "$FILE_PATH" ]; then
  exit 0
fi

# Get file extension
EXT="${FILE_PATH##*.}"

# Format based on file type using available formatters
case "$EXT" in
  ts|tsx|js|jsx|mjs|cjs)
    # Try prettier first, then biome, then deno fmt
    if command -v prettier &>/dev/null; then
      prettier --write "$FILE_PATH" 2>/dev/null || true
    elif command -v biome &>/dev/null; then
      biome format --write "$FILE_PATH" 2>/dev/null || true
    elif command -v deno &>/dev/null; then
      deno fmt "$FILE_PATH" 2>/dev/null || true
    fi
    ;;
  py)
    if command -v black &>/dev/null; then
      black --quiet "$FILE_PATH" 2>/dev/null || true
    elif command -v ruff &>/dev/null; then
      ruff format "$FILE_PATH" 2>/dev/null || true
    fi
    ;;
  rs)
    if command -v rustfmt &>/dev/null; then
      rustfmt "$FILE_PATH" 2>/dev/null || true
    fi
    ;;
  go)
    if command -v gofmt &>/dev/null; then
      gofmt -w "$FILE_PATH" 2>/dev/null || true
    fi
    ;;
  json)
    # Try prettier first, then biome, then jq as fallback
    if command -v prettier &>/dev/null; then
      prettier --write "$FILE_PATH" 2>/dev/null || true
    elif command -v biome &>/dev/null; then
      biome format --write "$FILE_PATH" 2>/dev/null || true
    elif command -v jq &>/dev/null; then
      TMP=$(mktemp)
      trap 'rm -f "$TMP"' EXIT
      if jq --indent 2 '.' "$FILE_PATH" > "$TMP" 2>/dev/null; then
        mv "$TMP" "$FILE_PATH"
      fi
      trap - EXIT
    fi
    ;;
  yaml|yml)
    # yamlfmt if available
    if command -v yamlfmt &>/dev/null; then
      yamlfmt "$FILE_PATH" 2>/dev/null || true
    fi
    ;;
  java)
    if command -v google-java-format &>/dev/null; then
      google-java-format --replace "$FILE_PATH" 2>/dev/null || true
    fi
    ;;
  rb)
    if command -v rubocop &>/dev/null; then
      rubocop -a --fail-level error "$FILE_PATH" 2>/dev/null || true
    fi
    ;;
  cs)
    if command -v dotnet &>/dev/null; then
      dotnet format --include "$FILE_PATH" 2>/dev/null || true
    fi
    ;;
  cpp|cc|cxx|h|hpp)
    if command -v clang-format &>/dev/null; then
      clang-format -i "$FILE_PATH" 2>/dev/null || true
    fi
    ;;
  ex|exs)
    if command -v mix &>/dev/null; then
      mix format "$FILE_PATH" 2>/dev/null || true
    fi
    ;;
  php)
    if command -v php-cs-fixer &>/dev/null; then
      php-cs-fixer fix "$FILE_PATH" --quiet 2>/dev/null || true
    fi
    ;;
  kt|kts)
    if command -v ktlint &>/dev/null; then
      ktlint -F "$FILE_PATH" 2>/dev/null || true
    fi
    ;;
  swift)
    if command -v swift-format &>/dev/null; then
      swift-format -i "$FILE_PATH" 2>/dev/null || true
    fi
    ;;
  scala)
    if command -v scalafmt &>/dev/null; then
      scalafmt "$FILE_PATH" 2>/dev/null || true
    fi
    ;;
esac

exit 0
