# Username Identity Contract Redesign (2026-02-18)

This package captures design plus implementation tracking for the multi-protocol identity cleanup discussed on 2026-02-18.

Files:
- `design.md` - target contract and rationale.
- `impact-surface-report.md` - implementation impact and risk analysis.
- `slice-plan.md` - proposed implementation sequence.
- `compatibility-note.md` - operator-visible behavior changes for implemented slices.
- `pr-invariants-checklist.md` - invariant verification for implemented slices.

Implementation status:
- Slice 1 + Slice 2 complete.
- Slice 3 complete (engine `UserMap`/`SetUserMap` removal with connector-local mapping ownership).
- Slice 3b complete (protocol-scoped bot identity state and `GetBotAttribute("id")` context rules).
- Slice 4 complete (SSH connector local identity schema via `ProtocolConfig.UserKeys` list entries).
- Slice 5 complete (Slack/Terminal/Test connector alignment to connector-local identity ownership).
- Slice 6 complete (ephemeral memory keying moved to username semantics with protocol-aware thread context).
- Slice 7 complete (final compatibility-bridge cleanup and contract finalization).

All planned slices in this package are now implemented.
