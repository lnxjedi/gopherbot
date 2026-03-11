#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

failed=0

report_fail() {
  echo "[docs-check] FAIL: $1" >&2
  failed=1
}

report_info() {
  echo "[docs-check] $1"
}

collect_active_docs() {
  find AGENTS.md aidocs devdocs \
    -type f -name '*.md' \
    ! -path 'aidocs/archive/*' \
    -print
}

report_info "checking stale path references in active docs"
if collect_active_docs | xargs rg -n "aidocs/GOALS_v3.md|devdocs/UPGRADING-v3.md|Upgrading_to_v3\.md"; then
  report_fail "found stale path reference(s); use root GOALS_v3.md and root UPGRADING-v3.md"
fi

report_info "checking stale implementation markers in active docs"
if collect_active_docs | xargs rg -n "commit pending|pending implementation|not yet started|: pending$"; then
  report_fail "found stale status marker(s) in active docs"
fi

report_info "checking devdocs index coverage"
actual_devdocs="$(find devdocs -maxdepth 1 -type f -name '*.md' -printf 'devdocs/%f\n' | sort | grep -v '^devdocs/README.md$' || true)"
declared_devdocs="$(rg -o "devdocs/[A-Za-z0-9._/-]+\.md" devdocs/README.md | sort -u || true)"
missing_in_index="$(comm -23 <(printf "%s\n" "$actual_devdocs") <(printf "%s\n" "$declared_devdocs") | sed '/^$/d' || true)"
if [[ -n "$missing_in_index" ]]; then
  echo "$missing_in_index" >&2
  report_fail "devdocs/README.md is missing one or more docs"
fi

report_info "checking active workstream index files"
for path in aidocs/multi-protocol/README.md aidocs/multiprocess/README.md aidocs/archive/README.md; do
  if [[ ! -f "$path" ]]; then
    report_fail "missing required index file: $path"
  fi
done

if [[ "$failed" -ne 0 ]]; then
  echo "[docs-check] one or more checks failed" >&2
  exit 1
fi

echo "[docs-check] OK"
