#!/usr/bin/env bash
#
# Generate a GitHub Actions job summary from golangci-lint JSON output.
#
# Reads JSON produced by golangci-lint --output.json.path and writes a
# markdown summary. Output always goes to stdout (visible in step logs)
# and to $GITHUB_STEP_SUMMARY when running in GitHub Actions.
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

# Write to a temp file, then copy to both stdout and summary
TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT

# Emit the temp file to stdout and $GITHUB_STEP_SUMMARY (if set)
emit() {
    cat "$TMP"
    if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
        cat "$TMP" >> "$GITHUB_STEP_SUMMARY"
    fi
}

# ---- guard ---------------------------------------------------------------

if [ ! -s "$INPUT_FILE" ]; then
    {
        echo "## Lint \`${LABEL}\`"
        echo ""
        echo "⚠️ No lint output captured (build may have failed)."
    } > "$TMP"
    emit
    exit 0
fi

# ---- parse ----------------------------------------------------------------

ISSUE_COUNT=$(jq '.Issues | length' "$INPUT_FILE")
LINTER_COUNT=$(jq '[.Report.Linters[] | select(.Enabled == true)] | length' "$INPUT_FILE")

if [ "$ISSUE_COUNT" -eq 0 ]; then
    {
        echo "## Lint \`${LABEL}\`"
        echo ""
        echo "✅ **0 issues** from ${LINTER_COUNT} linters"
    } > "$TMP"
    emit
    exit 0
fi

# ---- issues table ---------------------------------------------------------

{
    echo "## Lint \`${LABEL}\`"
    echo ""
    echo "| Linter | Location | Message |"
    echo "|---|---|---|"
    jq -r '.Issues[] | "| `\(.FromLinter)` | `\(.Pos.Filename):\(.Pos.Line)` | \(.Text) |"' "$INPUT_FILE"
    echo ""
    jq -r '
        [.Issues[] | .FromLinter] | group_by(.) |
        map({linter: .[0], count: length}) |
        sort_by(-.count)[] |
        "- **\(.linter)**: \(.count)"
    ' "$INPUT_FILE"
    echo ""
    echo "❌ **${ISSUE_COUNT} issues** from ${LINTER_COUNT} linters"
} > "$TMP"

emit
