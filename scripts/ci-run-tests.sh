#!/usr/bin/env bash
#
# Run Go tests and capture structured JSON output for CI reporting.
#
# Runs go test -json, writes NDJSON to OUTPUT_FILE, and shows human-readable
# output via jq. Exits non-zero if any tests fail or the build fails.
#
# Usage:
#   bash scripts/ci-run-tests.sh [output-file]
#
# Arguments:
#   output-file   Path for the NDJSON output (default: /tmp/test-output.json)
#
# Output files:
#   <output-file>          NDJSON events from go test -json (for summary script)
#   <output-file>.stderr   Captured stderr (build errors, etc.)

set -euo pipefail

OUTPUT_FILE="${1:-/tmp/test-output.json}"
STDERR_FILE="${OUTPUT_FILE}.stderr"
SENTINEL="${OUTPUT_FILE}.failed"

# Clean up any previous run artifacts.
rm -f "$SENTINEL"

# Run tests. Redirect stdout (JSON) and stderr (build errors) separately so
# build error text doesn't corrupt the JSON file. The || branch captures
# failure without triggering set -e so we can still display output first.
go test -json ./... >"$OUTPUT_FILE" 2>"$STDERR_FILE" || touch "$SENTINEL"

# Show build errors (compiler output, missing packages, etc.).
if [ -s "$STDERR_FILE" ]; then
    cat "$STDERR_FILE"
fi

# Show human-readable test output extracted from the JSON events.
if [ -s "$OUTPUT_FILE" ]; then
    jq -r 'select(.Action == "output") | .Output' "$OUTPUT_FILE" || true
fi

# Propagate test failure.
[ ! -f "$SENTINEL" ]
