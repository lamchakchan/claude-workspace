#!/usr/bin/env bash
#
# Generate a GitHub Actions job summary from go test -json output.
#
# Reads NDJSON produced by ci-run-tests.sh (or go test -json directly) and
# writes a markdown table of per-package results plus overall test counts.
# When run outside of GitHub Actions (no $GITHUB_STEP_SUMMARY), output goes
# to stdout so the script is testable locally.
#
# Usage:
#   bash scripts/ci-test-summary.sh [input-file] [label]
#
# Arguments:
#   input-file   NDJSON file from go test -json (default: /tmp/test-output.json)
#   label        Display label shown in the heading (default: local)
#
# Dependencies: jq

set -euo pipefail

INPUT_FILE="${1:-/tmp/test-output.json}"
LABEL="${2:-local}"
OUT="${GITHUB_STEP_SUMMARY:-/dev/stdout}"

# ---- guard ---------------------------------------------------------------

if [ ! -s "$INPUT_FILE" ]; then
    echo "## Tests \`${LABEL}\`" >> "$OUT"
    echo "" >> "$OUT"
    echo "⚠️ No test output captured (build may have failed)." >> "$OUT"
    exit 0
fi

# ---- per-package table ---------------------------------------------------

{
    echo "## Tests \`${LABEL}\`"
    echo ""
    echo "| Package | Tests | Result |"
    echo "|---|---|---|"
} >> "$OUT"

jq -rn '
[inputs] |
group_by(.Package)[] |
(map(select(.Test == null and (.Action == "pass" or .Action == "fail"))) | first) as $r |
select($r != null) |
{
  pkg:     (.[0].Package | split("/") | .[-2:] | join("/")),
  action:  $r.Action,
  elapsed: (($r.Elapsed // 0) | . * 100 | round | . / 100 | tostring | . + "s"),
  passed:  (map(select(.Test != null and .Action == "pass")) | length),
  failed:  (map(select(.Test != null and .Action == "fail")) | length)
} |
"| `\(.pkg)` | \(.passed + .failed) | \(if .action == "pass" then "✅ pass" else "❌ FAIL" end) (\(.elapsed)) |"
' "$INPUT_FILE" >> "$OUT"

# ---- totals --------------------------------------------------------------

TOTAL_PASS=$(jq -n '[inputs] | map(select(.Test != null and .Action == "pass")) | length' "$INPUT_FILE")
TOTAL_FAIL=$(jq -n '[inputs] | map(select(.Test != null and .Action == "fail")) | length' "$INPUT_FILE")
TOTAL_SKIP=$(jq -n '[inputs] | map(select(.Test != null and .Action == "skip")) | length' "$INPUT_FILE")

{
    echo ""
    if [ "$TOTAL_FAIL" -eq 0 ]; then
        echo "✅ **${TOTAL_PASS} passed**, ${TOTAL_SKIP} skipped"
    else
        echo "❌ **${TOTAL_FAIL} failed** — ${TOTAL_PASS} passed, ${TOTAL_SKIP} skipped"
    fi
} >> "$OUT"
