---
name: gopherbot-architecture-guardrails
description: Use when implementing or reviewing cross-cutting Gopherbot architecture changes that touch connectors, routing, startup/configuration, identity/authorization, or compatibility. Enforces strict pre-change impact analysis plus per-change invariants/testing/documentation checks.
---

# Gopherbot Architecture Guardrails

## When To Use

Use this skill for non-trivial changes that touch one or more of:

- Simultaneous multi-protocol runtime behavior
- Connector fan-in/fan-out routing
- Identity mapping and authorization decisions
- Startup flow, protocol selection, or configuration precedence
- Cross-connector failure isolation and ordering guarantees
- Robot bootstrap/setup workflow that affects startup, config loading, or connector semantics

Do not use this skill for small isolated plugin/job/task changes that do not affect connector, identity, routing, startup, or config behavior.

## Mandatory Preflight (Strict)

Before coding any non-trivial change, do all of the following:

1. Read (in order): `aidocs/README.md`, `aidocs/COMPONENT_MAP.md`, `aidocs/STARTUP_FLOW.md`, `aidocs/GOALS_v3.md`.
2. Review `aidocs/TESTING_CURRENT.md` for harness constraints.
3. Summarize: architectural invariants, startup ordering, connector assumptions, routing model, identity model.
4. Produce an Impact Surface Report using `references/impact-surface-report-template.md`.
5. Share the report in task context before implementation, unless explicitly waived by the user.

If waived, record the waiver in task context (example: "Impact report waived by user for this narrow change").

## Hybrid Working Model

This skill uses both:

- Checklist guardrails (always enforced)
- Light runbook phases (recommended order, adjustable with explicit rationale)

### Checklist Guardrails (Always)

- Shared authorization and business-policy logic stays in engine flows.
- Permission checks use protocol-agnostic username, not raw transport IDs.
- Connectors may use transport-specific internal user IDs, but must map IDs to shared username via roster.
- Cross-protocol identity equivalence must be explicit (never inferred heuristically).
- Per-connector message ordering guarantees must be preserved.
- Startup/control flow and config precedence must remain explicit and deterministic.
- When multiple connectors are enabled, connector failure isolation must prevent cascade failure.

### Recommended Phase Order (Runbook)

1. Identity and authorization substrate
2. Multi-connector runtime orchestration
3. Routing semantics and connector-local behavior boundaries
4. Startup/configuration/default behavior migration
5. Tests, documentation, and compatibility hardening

You may reorder phases when necessary, but state why in the impact report or task context.

## Required Artifacts Per Non-Trivial Change

Use these templates:

- Impact report: `references/impact-surface-report-template.md`
- PR invariants checklist: `references/pr-invariants-checklist-template.md`
- Compatibility note: `references/compatibility-note-template.md`

The compatibility note is required whenever behavior, config defaults, operator workflow, or externally visible semantics change.

## Execution Rhythm For Large Changes

Work in thin vertical slices:

1. Write impact report for one slice.
2. Implement only that slice.
3. Run focused tests first, then broader suite as needed.
4. Fill PR checklist and compatibility note.
5. Update affected `aidocs/` files in the same change.

Prefer multiple coherent PRs over one monolithic refactor.

## Testing And Documentation Gates

- Verify behavior against `aidocs/STARTUP_FLOW.md` if startup/config/order is touched.
- Update `aidocs/COMPONENT_MAP.md` for connector/module movement.
- Update connector-specific docs (`aidocs/SSH_CONNECTOR.md`, `aidocs/SLACK_CONNECTOR.md`, etc.) when semantics change.
- For test harness assumptions, verify with `aidocs/TESTING_CURRENT.md`.

## MCP Note

This skill assumes no MCP dependency for current work. Ignore MCP setup unless a future task explicitly requires MCP-backed tooling.

## Resources

- `references/impact-surface-report-template.md`
- `references/pr-invariants-checklist-template.md`
- `references/compatibility-note-template.md`
- `scripts/scaffold-change-docs.sh`
