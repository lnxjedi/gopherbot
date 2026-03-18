# Gopherbot — Gemini CLI Mandates

This file is the foundational mandate for Gemini CLI working in this repository. It takes absolute precedence over general defaults.

## Authority & Policies

- **This file is the authoritative operating procedure for AI agents working in this repository.** If any ad-hoc instruction conflicts with this file, this file wins.
- **Architectural Guardrails:** Non-trivial changes must follow the guardrails defined in this document and utilize the templates in `.gemini/skills/gopherbot-architecture-guardrails/references/`.

## Orientation Model (Tiered)

Before proposing or implementing changes, perform orientation as required:

### Tier A: Default Orientation (Required for all tasks)
1. Read `aidocs/README.md`.
2. Read `aidocs/COMPONENT_MAP.md`.
3. Read `GOALS_v3.md`.

Then load only the canonical docs needed for the task scope.

### Tier B: Escalated Orientation (Required if triggers apply)
Escalated orientation is mandatory if a change touches or may affect:
- Startup/config load order (`bot/start.go`, `bot/bot_process.go`, `bot/config_load.go`, `bot/conf.go`)
- Message routing/pipeline ordering (`bot/dispatch.go`, `bot/run_pipelines.go`, scheduler flow)
- Connector runtime/behavior (`connectors/*`, connector runtime orchestration)
- Identity/authz semantics (username mapping, roster gates, authorization/elevation)
- Root/default robot config structure (`conf/robot.yaml`, `robot.skel/conf/robot.yaml`)
- Cross-protocol behavior/contracts

**If uncertain, escalate.**

**Tier B Requirements:**
1. Read Tier A docs, plus `aidocs/STARTUP_FLOW.md` and `aidocs/TESTING_CURRENT.md`.
2. Summarize in your own words:
   - Core architectural invariants
   - Startup ordering constraints
   - Connector assumptions
   - Message routing model
   - Identity model

## Implementation Workflow

1. **Research & Strategy:** Use `grep_search` and `read_file` to validate assumptions.
2. **Impact Analysis:** For cross-cutting changes, produce an Impact Surface Report (template: `.gemini/skills/gopherbot-architecture-guardrails/references/impact-surface-report-template.md`) before modifying code. Do not implement until report is shared, unless explicitly waived by user.
3. **Execution:**
   - One logical change per branch/slice.
   - Update canonical docs in `aidocs/` in the same change (see Documentation Discipline).
   - Leverage `gopherbot-mcp` for robot interaction and testing (use `mcp_gopherbot-mcp_*` tools).
4. **Validation:**
   - Run `helpers/check-docs-hygiene.sh` for doc changes.
   - Run applicable integration tests (e.g., `test/bot_integration_test.go`).
   - Re-validate architectural invariants.

## Guardrails & Invariants

Unless explicitly updated in canonical docs, these must hold:
- Startup sequence is deterministic and traceable.
- Control flow is explicit, not implicit.
- Shared authorization/business policy remains in engine flows, not connectors.
- Permission/policy decisions are username-authoritative (protocol-agnostic).
- Message routing order is preserved within a connector.
- Configuration precedence is explicit and documented.
- Multi-connector isolation prevents cascading failure.
- Cross-protocol identity equivalence is canonical username, not heuristic transport-ID matching.

## Documentation Discipline

When behavior changes, update canonical docs in the same change:
- Startup/config loading/order: `aidocs/STARTUP_FLOW.md`
- Pipeline routing/execution ordering: `aidocs/PIPELINE_LIFECYCLE.md`
- Scheduled job behavior: `aidocs/SCHEDULER_FLOW.md`
- Connector behavior/identity mapping: Connector-specific docs (`aidocs/SSH_CONNECTOR.md`, `aidocs/SLACK_CONNECTOR.md`, etc.) and `aidocs/COMPONENT_MAP.md`.
- Execution security / privilege separation: `aidocs/EXECUTION_SECURITY_MODEL.md`
- Extension API/runtime semantics: `aidocs/EXTENSION_API.md` and/or `aidocs/EXTENSION_SURFACES.md`
- Compatibility/config migration: `aidocs/V3_COMPATIBILITY_CONTRACT.md` and root `UPGRADING-v3.md`.
- Test harness assumptions: `aidocs/TESTING_CURRENT.md`
