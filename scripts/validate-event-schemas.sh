#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "[event-schemas] Validating schema and sample JSON files..."

if command -v node >/dev/null 2>&1; then
  (cd "${ROOT_DIR}" && node scripts/validate-event-schemas.js)
elif command -v python3 >/dev/null 2>&1; then
  echo "[event-schemas] WARN: node is not installed; falling back to JSON parsing with python3."
  (cd "${ROOT_DIR}" && python3 - <<'PY'
import json
from pathlib import Path

root = Path.cwd()
files = list((root / "schemas" / "events").rglob("*.json"))
failures = 0
for file in files:
    try:
        json.loads(file.read_text())
    except Exception as exc:
        failures += 1
        print(f"[FAIL] {file.relative_to(root)} is not valid JSON: {exc}")

mapping_file = root / "schemas" / "events" / "sample-mapping.json"
if mapping_file.exists():
    mapping = json.loads(mapping_file.read_text())
    for sample, schema in mapping.items():
        if not (root / sample).exists():
            failures += 1
            print(f"[FAIL] Mapped sample is missing: {sample}")
        if not (root / schema).exists():
            failures += 1
            print(f"[FAIL] Mapped schema is missing: {schema}")
        if (root / sample).exists():
            json.loads((root / sample).read_text())

if failures:
    raise SystemExit(1)
print(f"[PASS] Parsed {len(files)} event schema JSON file(s). Ajv validation skipped.")
PY
  )
else
  echo "[event-schemas] FAIL: node or python3 is required to parse JSON."
  exit 1
fi
