# AGENTS.md — Gopherbot

This file defines the authoritative operating procedure for AI agents working in this repository.

It governs orientation, architectural invariants, planning requirements, and change discipline.

If any ad-hoc instruction conflicts with this file, this file wins.

---

## Phase 0 — Mandatory Orientation

Before proposing changes, you must read, in order:

1. `aidocs/README.md`
2. `aidocs/COMPONENT_MAP.md`
3. `aidocs/STARTUP_FLOW.md`
4. `aidocs/GOALS_v3.md`

For integration test harness and testbot behavior, see `aidocs/TESTING_CURRENT.md`.

Then summarize in your own words:

* Core architectural invariants
* Startup ordering constraints
* Connector assumptions
* Message routing model
* Identity model

Do not proceed until this summary is complete.

---

## Phase 1 — Impact Analysis (Required for Non-Trivial Changes)

For any change affecting connectors, routing, startup, configuration, or identity:

You must produce an Impact Surface Report before modifying code.

This report must include:

1. Subsystems affected (with file paths)
2. Current invariants that may break
3. Cross-cutting concerns (startup order, config loading, execution ordering)
4. Concurrency implications
5. Backward compatibility concerns
6. Documentation updates required

Do not implement non-trivial changes until this report is written and shared in the task context, unless explicitly waived by the user.

---

## Architectural Invariants

Unless explicitly updated in `aidocs/`, these invariants must hold:

* Startup sequence is deterministic and traceable.
* Control flow is explicit, not implicit.
* Shared authorization and business policy logic must remain in engine flows, not connectors.
* Identity resolution is deterministic and stable.
* Permission and policy decisions must be based on protocol-agnostic usernames, not raw transport IDs.
* Message routing order is preserved within a connector.
* Configuration precedence rules remain explicit and documented.
* When multiple connectors are enabled, cross-connector isolation prevents cascading failure.

If a change violates any invariant, update the documentation and explain why.

---

## Connector Rules (Critical for Multi-Protocol)

* Connectors own transport concerns and protocol-local interaction behavior.
* Connector-local behavior must not bypass shared authorization, policy, or business rules enforced by engine flows.
* Connectors must normalize engine-bound inbound messages into a canonical internal representation.
* Protocol-local commands and rendering are allowed when documented and isolated to that connector.
* Transport-specific internal user IDs are expected and allowed.
* Each protocol must map internal user IDs to a protocol-agnostic username via its roster.
* Cross-protocol identity equivalence must be explicit via roster mapping, never inferred heuristically.
* Replay buffers must define ordering guarantees explicitly.
* When multiple connectors are enabled, failure in one connector must not terminate others.

---

## Rules of Engagement

* Do not guess architecture. Trace behavior to concrete symbols.
* Cite file paths and functions.
* Prefer explicit state machines over implicit sequencing.
* When ambiguity exists, stop and present options with tradeoffs.
* For speculative behavior, insert `TODO (verify):`.

---

## Documentation Discipline

* Agent-facing system maps belong in `aidocs/`.
* Human development notes belong in `devdocs/`.
* Architectural changes require updating `aidocs/STARTUP_FLOW.md`.
* Connector changes require updating `aidocs/COMPONENT_MAP.md`.
* Identity or routing changes require documenting invariants explicitly.

Stale documentation is considered a defect.

---

## Change Hygiene

* One logical change per branch.
* Planning before implementation for cross-cutting changes.
* No silent refactors.
* Preserve behavior unless explicitly redefining it.
* If redefining behavior, document migration strategy.

---

## Post-Task Requirements

After implementation:

1. Re-validate architectural invariants.
2. Update all affected documentation.
3. Confirm startup sequence integrity.
4. Confirm connector isolation.
5. Confirm message ordering guarantees.
