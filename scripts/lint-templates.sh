#!/usr/bin/env bash
set -euo pipefail

# ---------- lint-templates.sh ----------
# Validates _template/ files against CUE schemas in lint/
#
# JSON files are validated directly. Markdown frontmatter is extracted
# into multi-document YAML streams so each category (agents, skills)
# is validated with a single cue vet call.

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LINT_DIR="$PROJECT_DIR/lint"
TEMPLATE_DIR="$PROJECT_DIR/_template"

# ---------- colors ----------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

# ---------- temp dir with cleanup ----------
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

# ---------- counters ----------
PASS_COUNT=0
FAIL_COUNT=0

# ---------- summary (GitHub Job Summary) ----------
SUMMARY=""
ERRORS=""

summary_row() {
    local category="$1" check="$2" result="$3"
    SUMMARY="${SUMMARY}| ${category} | ${check} | ${result} |"$'\n'
}

summary_errors() {
    local output="$1"
    ERRORS="${ERRORS}"$'\n'"<details><summary>Errors</summary>"$'\n'$'\n'"\`\`\`"$'\n'"${output}"$'\n'"\`\`\`"$'\n'$'\n'"</details>"$'\n'
}

# ---------- check cue is installed ----------
if ! command -v cue &>/dev/null; then
    if [[ "${CI:-}" == "true" ]]; then
        echo -e "${RED}ERROR:${NC} cue is not installed (required in CI)"
        echo "  Install: brew install cue-lang/tap/cue"
        exit 1
    else
        echo -e "${YELLOW}WARNING:${NC} cue is not installed — skipping template linting"
        echo "  Install: brew install cue-lang/tap/cue"
        exit 0
    fi
fi

# ---------- helpers ----------
vet() {
    local desc="$1"
    local category="$2"
    shift 2
    local output
    if output=$(cue vet "$@" 2>&1); then
        PASS_COUNT=$((PASS_COUNT + 1))
        echo -e "  ${GREEN}[PASS]${NC} $desc"
        summary_row "$category" "$desc" ":white_check_mark: pass"
    else
        FAIL_COUNT=$((FAIL_COUNT + 1))
        echo -e "  ${RED}[FAIL]${NC} $desc"
        echo "$output" | sed 's/^/         /'
        summary_row "$category" "$desc" ":x: FAIL"
        summary_errors "$output"
    fi
}

# extract_frontmatter — build a multi-document YAML stream from .md files
# Each file's YAML frontmatter becomes a separate --- document.
# A "# source: <filename>" comment is prepended for traceability.
# $1 = output file, $2 = category name (for summary), remaining = .md files
extract_frontmatter() {
    local output_file="$1"
    local category="$2"
    shift 2
    for md_file in "$@"; do
        local name
        name="$(basename "$md_file")"
        local frontmatter
        frontmatter=$(sed -n '/^---$/,/^---$/{ /^---$/d; p; }' "$md_file")
        if [[ -z "$frontmatter" ]]; then
            echo -e "  ${RED}[FAIL]${NC} $name (no frontmatter found)"
            FAIL_COUNT=$((FAIL_COUNT + 1))
            summary_row "$category" "$name (no frontmatter)" ":x: FAIL"
            continue
        fi
        printf -- "---\n# source: %s\n%s\n" "$name" "$frontmatter" >> "$output_file"
    done
}

# ---------- validate settings JSON ----------
echo -e "${BOLD}Validating settings...${NC}"

vet "project settings.json" "Settings" \
    "$TEMPLATE_DIR/project/.claude/settings.json" \
    "$LINT_DIR/settings.cue" \
    -d '#ProjectSettings'

vet "global settings.json" "Settings" \
    "$TEMPLATE_DIR/global/settings.json" \
    "$LINT_DIR/settings.cue" \
    -d '#GlobalSettings'

# Copy to .json temp file — cue requires a known extension
cp "$TEMPLATE_DIR/project/.claude/settings.local.json.example" "$TMPDIR/settings-local.json"
vet "settings.local.json.example" "Settings" \
    "$TMPDIR/settings-local.json" \
    "$LINT_DIR/settings.cue" \
    -d '#SettingsLocalExample'

# ---------- validate MCP config ----------
echo -e "\n${BOLD}Validating MCP config...${NC}"

vet "project .mcp.json" "MCP" \
    "$TEMPLATE_DIR/project/.mcp.json" \
    "$LINT_DIR/mcp.cue" \
    -d '#McpConfig'

# ---------- validate agent frontmatter ----------
echo -e "\n${BOLD}Validating agent definitions...${NC}"

AGENTS_YAML="$TMPDIR/agents.yaml"
extract_frontmatter "$AGENTS_YAML" "Agents" "$TEMPLATE_DIR/project/.claude/agents/"*.md

if [[ -s "$AGENTS_YAML" ]]; then
    AGENT_COUNT=$(grep -c '^---$' "$AGENTS_YAML")
    vet "agents ($AGENT_COUNT definitions)" "Agents" \
        "$AGENTS_YAML" \
        "$LINT_DIR/agents.cue" \
        -d '#AgentFrontmatter'
fi

# ---------- validate skill frontmatter ----------
echo -e "\n${BOLD}Validating skill definitions...${NC}"

SKILLS_YAML="$TMPDIR/skills.yaml"
extract_frontmatter "$SKILLS_YAML" "Skills" "$TEMPLATE_DIR/project/.claude/skills/"*/SKILL.md

if [[ -s "$SKILLS_YAML" ]]; then
    SKILL_COUNT=$(grep -c '^---$' "$SKILLS_YAML")
    vet "skills ($SKILL_COUNT definitions)" "Skills" \
        "$SKILLS_YAML" \
        "$LINT_DIR/skills.cue" \
        -d '#SkillFrontmatter'
fi

# ---------- summary ----------
echo ""
TOTAL=$((PASS_COUNT + FAIL_COUNT))
echo -e "${BOLD}Results: ${GREEN}${PASS_COUNT} passed${NC}, ${RED}${FAIL_COUNT} failed${NC} (${TOTAL} total)"

# ---------- write GitHub Job Summary ----------
if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
    {
        echo "## Lint templates"
        echo ""
        echo "| Category | Check | Result |"
        echo "|---|---|---|"
        printf '%s' "$SUMMARY"
        if [[ -n "$ERRORS" ]]; then
            printf '%s' "$ERRORS"
        fi
        echo ""
        if [[ $FAIL_COUNT -eq 0 ]]; then
            echo ":white_check_mark: **${PASS_COUNT} passed**"
        else
            echo ":x: **${FAIL_COUNT} failed** — ${PASS_COUNT} passed"
        fi
    } >> "$GITHUB_STEP_SUMMARY"
fi

if [[ $FAIL_COUNT -gt 0 ]]; then
    exit 1
fi
