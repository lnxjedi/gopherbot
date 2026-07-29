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
  {
    find . -maxdepth 1 -type f -name '*.md' -print
    find aidocs devdocs -type f -name '*.md' \
      ! -path 'aidocs/archive/*' -print
    if [[ -f aidocs/archive/README.md ]]; then
      printf '%s\n' aidocs/archive/README.md
    fi
  } | sed 's#^\./##' | sort -u
}

validate_reference() {
  local source="$1"
  local line="$2"
  local reference="${3%%#*}"
  local source_dir

  [[ "$reference" == *"<"* || "$reference" == *">"* ]] && return
  source_dir="$(dirname "$source")"
  if [[ ! -f "$reference" && ! -f "$source_dir/$reference" ]]; then
    report_fail "$source:$line references missing document: $reference"
  fi
}

report_info "checking local document references"
while IFS= read -r source; do
  while IFS=: read -r line match; do
    reference="${match#\`}"
    validate_reference "$source" "$line" "${reference%\`}"
  done < <(rg -n -o '`[A-Za-z0-9._<>/-]+\.md`' "$source" || true)

  while IFS=: read -r line match; do
    reference="${match#](}"
    validate_reference "$source" "$line" "${reference%)}"
  done < <(rg -n -o '\]\([A-Za-z0-9._<>/-]+\.md(#[A-Za-z0-9._-]+)?\)' "$source" || true)
done < <(collect_active_docs)

check_index_coverage() {
  local directory="$1"
  local index="$directory/README.md"
  local actual
  local declared
  local missing

  actual="$(find "$directory" -maxdepth 1 -type f -name '*.md' \
    ! -name 'README.md' -print | sort)"
  declared="$(rg -o '`[A-Za-z0-9._/-]+\.md`' "$index" \
    | tr -d '`' \
    | while IFS= read -r reference; do
        if [[ "$reference" == "$directory/"* ]]; then
          printf '%s\n' "$reference"
        else
          printf '%s\n' "$directory/$reference"
        fi
      done \
    | sort -u || true)"
  missing="$(comm -23 \
    <(printf '%s\n' "$actual") \
    <(printf '%s\n' "$declared") \
    | sed '/^$/d' || true)"

  if [[ -n "$missing" ]]; then
    printf '%s\n' "$missing" >&2
    report_fail "$index does not list every top-level document"
  fi
}

report_info "checking documentation index coverage"
check_index_coverage aidocs
check_index_coverage devdocs

if [[ "$failed" -ne 0 ]]; then
  echo "[docs-check] one or more checks failed" >&2
  exit 1
fi

echo "[docs-check] OK"
