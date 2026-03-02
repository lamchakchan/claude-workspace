#!/bin/bash
set -euo pipefail

# Scans file content being written for potential secrets
INPUT=$(cat)
CONTENT=$(echo "$INPUT" | jq -r '.tool_input.content // .tool_input.new_string // empty')
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

if [ -z "$CONTENT" ]; then
  exit 0
fi

# Skip if writing to allowed placeholder file types (intentional examples)
if echo "$FILE_PATH" | grep -qE '\.(example|template|sample)$'; then
  exit 0
fi

# Check for common secret patterns
PATTERNS=(
  'AKIA[0-9A-Z]{16}'                          # AWS Access Key
  'sk-[a-zA-Z0-9]{20,}'                       # OpenAI/Stripe-style keys
  'sk-ant-[a-zA-Z0-9-]{20,}'                  # Anthropic API keys
  'ghp_[a-zA-Z0-9]{36}'                       # GitHub Personal Access Token
  'gho_[a-zA-Z0-9]{36}'                       # GitHub OAuth Token
  'glpat-[a-zA-Z0-9_-]{20,}'                  # GitLab Personal Access Token
  'xox[bpors]-[a-zA-Z0-9-]+'                  # Slack tokens
  '-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----'   # Private keys
  'password\s*[:=]\s*["\x27][^"\x27]{8,}'     # Hardcoded passwords
  'secret\s*[:=]\s*["\x27][^"\x27]{8,}'       # Hardcoded secrets
  '"type"\s*:\s*"service_account"'             # GCP service account key
  '"clientSecret"\s*:\s*"[^"]+'               # Azure AD client secret
  '(password|pwd)\s*=\s*[^;]{8,}'             # DB connection string passwords
  '(jwt_secret|JWT_SECRET)\s*[:=]\s*["\x27][^"\x27]{8,}'  # JWT secrets
  'AZURE_[A-Z_]*SECRET\s*[:=]\s*["\x27][^"\x27]{8,}'      # Azure secrets
  'SG\.[a-zA-Z0-9_-]{22}\.[a-zA-Z0-9_-]{43}'              # SendGrid API key
  'sq0[a-z]{3}-[a-zA-Z0-9_-]{22,}'                         # Square API key
)

for PATTERN in "${PATTERNS[@]}"; do
  if echo "$CONTENT" | grep -qEi "$PATTERN"; then
    echo "Blocked: Potential secret or credential detected in file content. Pattern: $PATTERN" >&2
    echo "If this is intentional (e.g., a regex pattern or example), use a placeholder value instead." >&2
    exit 2
  fi
done

exit 0
