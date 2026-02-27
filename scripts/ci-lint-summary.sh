#!/usr/bin/env bash
#
# Generate a GitHub Actions job summary from golangci-lint JSON output.
#
# Reads JSON produced by golangci-lint --output.json.path and writes a
# markdown summary of lint results. When run outside of GitHub Actions
# (no $GITHUB_STEP_SUMMARY), output goes to stdout so the script is
# testable locally.
#
# Usage:
#   bash scripts/ci-lint-summary.sh [input-file] [label]
#
# Arguments:
#   input-file   JSON file from golangci-lint (default: /tmp/lint-output.json)
#   label        Display label shown in the heading (default: local)
#
# Dependencies: jq

set -euo pipefail

INPUT_FILE="${1:-/tmp/lint-output.json}"
LABEL="${2:-local}"
OUT="${GITHUB_STEP_SUMMARY:-/dev/stdout}"

# ---- guard ---------------------------------------------------------------

# Log file status to step logs (stderr) for diagnostics
ls -la "$INPUT_FILE" >&2 2>/dev/null || echo "ci-lint-summary: $INPUT_FILE not found" >&2

if [ ! -s "$INPUT_FILE" ]; then
    echo "## Lint \`${LABEL}\`" >> "$OUT"
    echo "" >> "$OUT"
    echo "⚠️ No lint output captured (build may have failed)." >> "$OUT"
    exit 0
fi

# ---- parse ----------------------------------------------------------------

ISSUE_COUNT=$(jq '.Issues | length' "$INPUT_FILE")
LINTER_COUNT=$(jq '[.Report.Linters[] | select(.Enabled == true)] | length' "$INPUT_FILE")

if [ "$ISSUE_COUNT" -eq 0 ]; then
    echo "## Lint \`${LABEL}\`" >> "$OUT"
    echo "" >> "$OUT"
    echo "✅ **0 issues** from ${LINTER_COUNT} linters" >> "$OUT"
    exit 0
fi

# ---- issues table ---------------------------------------------------------

{
    echo "## Lint \`${LABEL}\`"
    echo ""
    echo "| Linter | Location | Message |"
    echo "|---|---|---|"
} >> "$OUT"

jq -r '.Issues[] | "| `\(.FromLinter)` | `\(.Pos.Filename):\(.Pos.Line)` | \(.Text) |"' \
    "$INPUT_FILE" >> "$OUT"

# ---- per-linter breakdown ------------------------------------------------

{
    echo ""
    jq -r '
        [.Issues[] | .FromLinter] | group_by(.) |
        map({linter: .[0], count: length}) |
        sort_by(-.count)[] |
        "- **\(.linter)**: \(.count)"
    ' "$INPUT_FILE"
    echo ""
    echo "❌ **${ISSUE_COUNT} issues** from ${LINTER_COUNT} linters"
} >> "$OUT"
