#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <change-slug>"
  echo "Example: $0 identity-roster-phase1"
  exit 1
fi

slug="$(echo "$1" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g; s/^-+|-+$//g; s/-+/-/g')"
if [[ -z "$slug" ]]; then
  echo "error: slug resolved to empty value"
  exit 1
fi

if [[ ! -d "aidocs" ]]; then
  echo "error: run from the repository root (aidocs/ not found)"
  exit 1
fi

skill_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
today="$(date +%F)"
out_dir="aidocs/multi-protocol/${today}-${slug}"

mkdir -p "$out_dir"
cp "${skill_dir}/references/impact-surface-report-template.md" "${out_dir}/impact-surface-report.md"
cp "${skill_dir}/references/pr-invariants-checklist-template.md" "${out_dir}/pr-invariants-checklist.md"
cp "${skill_dir}/references/compatibility-note-template.md" "${out_dir}/compatibility-note.md"

echo "created:"
echo "  ${out_dir}/impact-surface-report.md"
echo "  ${out_dir}/pr-invariants-checklist.md"
echo "  ${out_dir}/compatibility-note.md"

