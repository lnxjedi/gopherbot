# Provider Config Layout Refactor (2026-02-18)

This package captures planning artifacts for moving provider-specific configuration out of `conf/robot.yaml` into provider-scoped config files.

Primary design target:
- move brain provider configuration to `conf/brains/<provider>.yaml`
- move history provider configuration to `conf/history/<provider>.yaml`
- keep environment files focused on provider selection and runtime defaults (for example `Brain: cloudflare`), not provider-specific credential blocks

Files:
- `design.md` - proposed contract and rationale.
- `impact-surface-report.md` - impact analysis and risks.
- `slice-plan.md` - incremental implementation slices.
- `compatibility-note.md` - migration contract and operator-facing expectations.
- `pr-invariants-checklist.md` - implementation verification checklist.

Status:
- Slice 1 complete (engine provider-file loading + validation contract).
- Slice 2 complete (default `conf/` provider-file migration).
- Slice 3 complete (`robot.skel` provider-file migration).
- Slice 4 complete (Clu provider-file + environment/deploy migration).
- Slice 5 complete (integration test fixture migration for provider-file contract).
- Slice 6 complete (docs + upgrading contract finalization).
