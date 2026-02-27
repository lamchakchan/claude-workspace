#!/usr/bin/env bash
#
# Shared test phase functions for dev-env.sh and smoke-test.sh.
#
# Expects the caller to have already sourced lib.sh (for vm_exec,
# vm_exec_quiet, assert, assert_pass, assert_fail, color variables).
#
# Convention: run_phase_attach sets the global PROJECT variable, which
# is used by run_phase_doctor, run_phase_sessions, run_phase_upgrade_check.
#
# Hook functions (optional, defined by the caller):
#   phase_setup_extra    — called at end of run_phase_setup for extra assertions
#
# Overridable functions:
#   _phase_skip <msg>    — called when a test is skipped (e.g., rate-limited API)
#
# Functions:
#   run_phase_setup
#   run_phase_attach
#   run_phase_doctor
#   run_phase_sessions
#   run_phase_upgrade_check

# Default skip handler: prints yellow SKIP message.
# Callers can override by defining their own _phase_skip function.
_phase_skip() {
    local msg="$1"
    echo -e "  ${YELLOW}[SKIP]${NC} $msg"
}

run_phase_setup() {
    echo -e "\n${BOLD}--- Phase: claude-workspace setup ---${NC}"

    local output
    output=$(vm_exec "claude-workspace setup" 2>&1) || true
    echo "$output" | sed 's/^/  | /'

    echo ""
    echo "  Assertions:"

    assert "~/.claude/settings.json exists" \
        vm_exec_quiet "test -f /home/ubuntu/.claude/settings.json"

    assert "~/.claude/CLAUDE.md exists" \
        vm_exec_quiet "test -f /home/ubuntu/.claude/CLAUDE.md"

    assert "claude-workspace is executable in PATH" \
        vm_exec_quiet "test -x /usr/local/bin/claude-workspace"

    # Hook: caller can define phase_setup_extra for additional assertions
    if type -t phase_setup_extra &>/dev/null; then
        phase_setup_extra
    fi
}

run_phase_attach() {
    echo -e "\n${BOLD}--- Phase: claude-workspace attach ---${NC}"

    echo "  Creating test project..."
    vm_exec "mkdir -p /home/ubuntu/test-project && cd /home/ubuntu/test-project && git init && git config user.email test@example.com && git config user.name Test && touch README.md && git add . && git commit -m 'init' -q"

    local output
    output=$(vm_exec "claude-workspace attach /home/ubuntu/test-project" 2>&1) || true
    echo "$output" | sed 's/^/  | /'

    echo ""
    echo "  Assertions:"

    # Set global PROJECT for use by subsequent phases
    PROJECT="/home/ubuntu/test-project"

    assert ".claude/settings.json exists" \
        vm_exec_quiet "test -f ${PROJECT}/.claude/settings.json"

    assert ".claude/CLAUDE.md exists" \
        vm_exec_quiet "test -f ${PROJECT}/.claude/CLAUDE.md"

    assert ".mcp.json exists and is valid JSON" \
        vm_exec_quiet "python3 -c \"import json; json.load(open('${PROJECT}/.mcp.json'))\""

    for hook in auto-format.sh block-dangerous-commands.sh enforce-branch-policy.sh validate-secrets.sh; do
        assert "hook ${hook} exists and is executable" \
            vm_exec_quiet "test -x ${PROJECT}/.claude/hooks/${hook}"
    done

    assert ".claude/agents/ is non-empty" \
        vm_exec_quiet "test -d ${PROJECT}/.claude/agents && [ \"\$(ls -A ${PROJECT}/.claude/agents)\" ]"

    assert ".claude/skills/ is non-empty" \
        vm_exec_quiet "test -d ${PROJECT}/.claude/skills && [ \"\$(ls -A ${PROJECT}/.claude/skills)\" ]"

    # Gitignore assertions
    assert ".claude/.gitignore exists" \
        vm_exec_quiet "test -f ${PROJECT}/.claude/.gitignore"

    assert ".claude/.gitignore contains MEMORY.md" \
        vm_exec_quiet "grep -q 'MEMORY.md' ${PROJECT}/.claude/.gitignore"

    assert ".claude/.gitignore contains *.jsonl" \
        vm_exec_quiet "grep -q '\*.jsonl' ${PROJECT}/.claude/.gitignore"

    assert ".claude/.gitignore contains audits/" \
        vm_exec_quiet "grep -q 'audits/' ${PROJECT}/.claude/.gitignore"

    assert "root .gitignore exists" \
        vm_exec_quiet "test -f ${PROJECT}/.gitignore"

    assert "root .gitignore contains plans/*.md" \
        vm_exec_quiet "grep -q 'plans/\*.md' ${PROJECT}/.gitignore"

    assert "root .gitignore contains !plans/.gitkeep" \
        vm_exec_quiet "grep -q '!plans/.gitkeep' ${PROJECT}/.gitignore"

    assert "plans/.gitkeep exists" \
        vm_exec_quiet "test -f ${PROJECT}/plans/.gitkeep"
}

run_phase_doctor() {
    echo -e "\n${BOLD}--- Phase: claude-workspace doctor ---${NC}"

    local output
    output=$(vm_exec "cd ${PROJECT} && ANTHROPIC_API_KEY=sk-fake claude-workspace doctor" 2>&1) || true
    echo "$output" | sed 's/^/  | /'

    echo ""
    echo "  Assertions:"

    local fail_lines
    fail_lines=$(echo "$output" | grep -c '\[FAIL\]' || true)
    if [[ "$fail_lines" -eq 0 ]]; then
        assert_pass "doctor output contains no [FAIL] lines"
    else
        assert_fail "doctor output contains ${fail_lines} [FAIL] line(s)"
    fi
}

run_phase_sessions() {
    echo -e "\n${BOLD}--- Phase: claude-workspace sessions ---${NC}"

    # Create fake session data
    echo "  Creating test session data..."
    vm_exec 'mkdir -p /home/ubuntu/.claude/projects/-home-ubuntu-test-project'
    vm_exec 'cat > /home/ubuntu/.claude/projects/-home-ubuntu-test-project/aaaaaaaa-1111-2222-3333-444444444444.jsonl << '\''SESS'\''
{"type":"file-history-snapshot","messageId":"snap-1"}
{"type":"user","message":{"role":"user","content":"Add authentication middleware"},"timestamp":"2026-02-24T10:00:00.000Z","cwd":"/home/ubuntu/test-project","uuid":"u1","parentUuid":null,"isMeta":false}
{"type":"user","message":{"role":"user","content":"Also add rate limiting"},"timestamp":"2026-02-24T10:05:00.000Z","cwd":"/home/ubuntu/test-project","uuid":"u2","parentUuid":"u1","isMeta":false}
SESS'
    vm_exec 'cat > /home/ubuntu/.claude/projects/-home-ubuntu-test-project/bbbbbbbb-5555-6666-7777-888888888888.jsonl << '\''SESS'\''
{"type":"file-history-snapshot","messageId":"snap-2"}
{"type":"user","message":{"role":"user","content":"<command-name>/exit</command-name>"},"timestamp":"2026-02-24T09:00:00.000Z","cwd":"/home/ubuntu/test-project","uuid":"u3","parentUuid":null,"isMeta":false}
SESS'

    echo ""
    echo "  Assertions:"

    local sessions_list
    sessions_list=$(vm_exec "cd ${PROJECT} && claude-workspace sessions" 2>&1) || true
    echo "$sessions_list" | sed 's/^/  | /'

    if echo "$sessions_list" | grep -q 'Add authentication middleware'; then
        assert_pass "sessions list shows session with real prompt"
    else
        assert_fail "sessions list shows session with real prompt"
    fi

    if echo "$sessions_list" | grep -q 'bbbbbbbb'; then
        assert_fail "sessions list filters out empty sessions (exit-only)"
    else
        assert_pass "sessions list filters out empty sessions (exit-only)"
    fi

    echo ""
    local sessions_show
    sessions_show=$(vm_exec "cd ${PROJECT} && claude-workspace sessions show aaaaaaaa" 2>&1) || true
    echo "$sessions_show" | sed 's/^/  | /'

    if echo "$sessions_show" | grep -q 'Add authentication middleware'; then
        assert_pass "sessions show displays first prompt"
    else
        assert_fail "sessions show displays first prompt"
    fi

    if echo "$sessions_show" | grep -q 'Also add rate limiting'; then
        assert_pass "sessions show displays second prompt"
    else
        assert_fail "sessions show displays second prompt"
    fi

    if echo "$sessions_show" | grep -q 'Prompts: 2'; then
        assert_pass "sessions show reports correct prompt count"
    else
        assert_fail "sessions show reports correct prompt count"
    fi
}

run_phase_upgrade_check() {
    echo -e "\n${BOLD}--- Phase: claude-workspace upgrade --check ---${NC}"

    local exit_code=0
    local output
    output=$(vm_exec "claude-workspace upgrade --check" 2>&1) || exit_code=$?
    echo "$output" | sed 's/^/  | /'

    echo ""
    echo "  Assertions:"

    if [[ "$exit_code" -eq 1 ]]; then
        assert_pass "exit code is 1 (update available)"
    else
        if echo "$output" | grep -q "rate limit\|checking for updates"; then
            _phase_skip "GitHub API unavailable (rate limited or no network)"
        else
            assert_fail "exit code was ${exit_code}, expected 1 (update available)"
        fi
    fi

    if echo "$output" | grep -q "Current: dev"; then
        assert_pass "output shows 'Current: dev'"
    else
        assert_fail "output missing 'Current: dev'"
    fi

    if echo "$output" | grep -q "Latest:"; then
        assert_pass "output shows 'Latest:' version"
    else
        if echo "$output" | grep -q "rate limit\|checking for updates"; then
            _phase_skip "Cannot verify latest version (API unavailable)"
        else
            assert_fail "output missing 'Latest:' version"
        fi
    fi

    if echo "$output" | grep -q "dev build"; then
        assert_pass "output shows dev build warning"
    else
        if echo "$output" | grep -q "rate limit\|checking for updates"; then
            _phase_skip "Cannot verify dev warning (API unavailable)"
        else
            assert_fail "output missing dev build warning"
        fi
    fi
}
