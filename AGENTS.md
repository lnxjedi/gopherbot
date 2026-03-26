# AGENTS.md — Gopherbot

This file is the authoritative operating procedure for AI agents working in this repository.

If any ad-hoc instruction conflicts with this file, this file wins.

## Authority Boundary

- `AGENTS.md` is the single source of policy and required process.
- Skills may provide workflows/templates, but must not redefine repository policy.

## Phase 0 — Orientation (Two-Tier Model)

### Tier A: Default Orientation (required for all tasks)

Before proposing or implementing changes, read:
1. `aidocs/README.md`
2. `aidocs/COMPONENT_MAP.md`
3. root `GOALS_v3.md`

Then load only the canonical docs needed for the task scope.

### Tier B: Escalated Orientation (hard requirement when triggered)

You must run full architecture preflight before coding when **any** trigger applies.

Read in order:
1. `aidocs/README.md`
2. `aidocs/COMPONENT_MAP.md`
3. `aidocs/STARTUP_FLOW.md`
4. root `GOALS_v3.md`
5. `aidocs/TESTING_CURRENT.md`

Then summarize in your own words:
- core architectural invariants
- startup ordering constraints
- connector assumptions
- message routing model
- identity model

### Escalation Triggers (hard)

Escalated orientation is mandatory if a change touches or may affect:
- startup/config load order (`bot/start.go`, `bot/bot_process.go`, `bot/config_load.go`, `bot/conf.go`)
- message routing/pipeline ordering (`bot/dispatch.go`, `bot/run_pipelines.go`, scheduler flow)
- connector runtime/behavior (`connectors/*`, connector runtime orchestration)
- identity/authz semantics (username mapping, roster gates, authorization/elevation)
- root/default robot config structure (`conf/robot.yaml`, `robot.skel/conf/robot.yaml`)
- cross-protocol behavior/contracts

If uncertain, escalate.

## Phase 1 — Impact Analysis (Required for Cross-Cutting Changes)

For changes affecting connectors, routing, startup, configuration, identity, or compatibility:
- produce an Impact Surface Report before modifying code
- include subsystems, invariants, cross-cutting concerns, concurrency, compatibility, docs updates
- do not implement until report is shared, unless explicitly waived by user

## Architectural Invariants

Unless explicitly updated in canonical docs, these must hold:
- startup sequence is deterministic and traceable
- control flow is explicit, not implicit
- shared authorization/business policy remains in engine flows, not connectors
- permission/policy decisions are username-authoritative
- message routing order is preserved within a connector
- configuration precedence is explicit and documented
- engine-shipped extension defaults remain authoritative; custom robot extension config stays delta-only unless behavior is intentionally redefined
- multi-connector isolation prevents cascading failure
- secret access is explicit and scope-based: unprivileged extensions must not discover shared secrets through generic robot methods

## Connector Rules (Critical for Multi-Protocol)

- connectors own transport concerns and protocol-local behavior
- connectors must not bypass shared engine policy/authorization logic
- connectors map transport identity to canonical username deterministically
- cross-protocol identity equivalence is canonical username, not heuristic transport-ID matching
- connector failure isolation must be preserved when multiple connectors are enabled

## Extension Secret Boundary

- secrets may be exposed to an extension only through explicit administrator configuration for that extension, or through memory/brain state owned by that extension's authorized namespace
- unprivileged robot methods must not reveal shared secret-bearing configuration, nor provide indirect discovery of secrets outside the caller's granted scope
- do not add or document extension APIs that return provider registries, parameter-set contents, or other broad configuration objects containing secrets

## Documentation Discipline (Hard Mapping)

When behavior changes, update canonical docs in the same change:

- startup/config loading/order
  - `aidocs/STARTUP_FLOW.md`
- pipeline routing/execution ordering
  - `aidocs/PIPELINE_LIFECYCLE.md`
- scheduled job behavior
  - `aidocs/SCHEDULER_FLOW.md`
- connector behavior/identity mapping
  - connector-specific docs (`aidocs/SSH_CONNECTOR.md`, `aidocs/SLACK_CONNECTOR.md`, etc.)
  - `aidocs/COMPONENT_MAP.md` if component ownership/boundaries moved
- execution security / privilege separation behavior
  - `aidocs/EXECUTION_SECURITY_MODEL.md`
- extension API/runtime semantics
  - `aidocs/EXTENSION_API.md` and/or `aidocs/EXTENSION_SURFACES.md`
- compatibility/config migration behavior
  - `aidocs/V3_COMPATIBILITY_CONTRACT.md`
  - root `UPGRADING-v3.md`
  - corresponding defaults/templates in `conf/` and `robot.skel/`
- test harness assumptions
  - `aidocs/TESTING_CURRENT.md`

Additional rules:
- archive docs under `aidocs/archive/` are reference-only and non-canonical
- roadmap remains root `GOALS_v3.md` (human-maintained)
- AI "what's left" backlog remains `aidocs/TODO.md` (AI-maintained)

## Docs Hygiene Gate

For any change touching `aidocs/`, `devdocs/`, `AGENTS.md`, or `UPGRADING-v3.md`:
- run `helpers/check-docs-hygiene.sh`
- fix reported stale references/markers before completion

Stale documentation is a defect.

## v3 Compatibility Contract

Follow `aidocs/V3_COMPATIBILITY_CONTRACT.md`.

Required stance:
- extension API behavior compatibility is priority
- username-based security behavior is priority
- persistent brain compatibility is prioritized where feasible
- config backward compatibility is not required; migration is acceptable when documented

## Change Hygiene

- one logical change per branch
- planning before implementation for cross-cutting changes
- no silent refactors
- preserve behavior unless explicitly redefining it
- if redefining behavior, document migration strategy

## Post-Task Requirements

After implementation:
1. re-validate architectural invariants
2. update all required canonical docs
3. confirm startup sequence integrity
4. confirm connector isolation and ordering guarantees
5. run applicable tests and docs hygiene checks

For any task where integration tests are applicable:
6. run the applicable integration suite before closing out work
7. classify every integration failure as either:
   - a real regression / newly introduced bug
   - an intentional behavior change with outdated test expectations
8. do not update test expectations until that classification is explicit
