#!/bin/bash
set -euo pipefail

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
    if command -v jq &>/dev/null; then
      TMP=$(mktemp)
      if jq '.' "$FILE_PATH" > "$TMP" 2>/dev/null; then
        mv "$TMP" "$FILE_PATH"
      else
        rm -f "$TMP"
      fi
    fi
    ;;
  yaml|yml)
    # yamlfmt if available
    if command -v yamlfmt &>/dev/null; then
      yamlfmt "$FILE_PATH" 2>/dev/null || true
    fi
    ;;
esac

exit 0
