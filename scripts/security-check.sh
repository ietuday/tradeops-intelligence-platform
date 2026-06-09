#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

PASS_COUNT=0
WARN_COUNT=0
FAIL_COUNT=0

pass() {
  PASS_COUNT=$((PASS_COUNT + 1))
  printf 'PASS %s\n' "$1"
}

warn() {
  WARN_COUNT=$((WARN_COUNT + 1))
  printf 'WARN %s\n' "$1"
}

fail() {
  FAIL_COUNT=$((FAIL_COUNT + 1))
  printf 'FAIL %s\n' "$1"
}

tracked_files() {
  if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    git ls-files
  else
    find . -type f | sed 's#^\./##'
  fi
}

RISKY_FILES="$(tracked_files | grep -E '(^|/)(\.env|.*\.pem|.*\.key|id_rsa|private_key|infrastructure/docker/\.env)$' | grep -vE '(^|/)\.env\.example$' || true)"
if [ -n "${RISKY_FILES}" ]; then
  fail "risky secret-bearing filenames are tracked:"
  printf '%s\n' "${RISKY_FILES}"
else
  pass "no tracked .env/private key files found"
fi

SECRET_MATCHES="$(rg -n --hidden -S '(JWT_SECRET=|PASSWORD=|API_KEY=|PRIVATE_KEY|AWS_SECRET_ACCESS_KEY)' . \
  --glob '!.git/**' \
  --glob '!node_modules/**' \
  --glob '!dist/**' \
  --glob '!coverage/**' \
  --glob '!.venv/**' \
  --glob '!**/__pycache__/**' \
  --glob '!**/.pytest_cache/**' \
  --glob '!infrastructure/docker/.env.example' \
  --glob '!docs/examples/**' || true)"

if [ -n "${SECRET_MATCHES}" ]; then
  warn "secret-like strings found; review that they are placeholders or variable references:"
  printf '%s\n' "${SECRET_MATCHES}"
else
  pass "no suspicious secret-like strings found outside allowed examples"
fi

if rg -n "helmet\\(" services/api-gateway/src >/dev/null 2>&1; then
  pass "API Gateway Helmet middleware detected"
else
  fail "API Gateway Helmet middleware not detected"
fi

if rg -n "createRateLimitMiddleware|RATE_LIMIT_MAX_REQUESTS|RATE_LIMIT_WINDOW_MS" services/api-gateway/src services/api-gateway/test >/dev/null 2>&1; then
  pass "API Gateway rate limiting config detected"
else
  fail "API Gateway rate limiting config not detected"
fi

if rg -n "REQUEST_BODY_LIMIT|express\\.json\\(\\{ limit" services/api-gateway/src services/api-gateway/test >/dev/null 2>&1; then
  pass "API Gateway request body limit config detected"
else
  fail "API Gateway request body limit config not detected"
fi

if rg -n "CORS_ORIGIN|cors\\(" services/api-gateway/src infrastructure/docker/docker-compose.yml >/dev/null 2>&1; then
  pass "API Gateway CORS config detected"
else
  warn "API Gateway CORS config not detected"
fi

printf '\nSecurity check summary: %s PASS, %s WARN, %s FAIL\n' "${PASS_COUNT}" "${WARN_COUNT}" "${FAIL_COUNT}"

if [ "${FAIL_COUNT}" -gt 0 ]; then
  exit 1
fi
