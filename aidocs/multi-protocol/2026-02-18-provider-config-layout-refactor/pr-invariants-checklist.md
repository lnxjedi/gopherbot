# PR Invariants Checklist

## Scope

- PR slice: provider-config-layout-refactor
- Linked impact report: `aidocs/multi-protocol/2026-02-18-provider-config-layout-refactor/impact-surface-report.md`

## Core Invariants

- Startup sequence remains deterministic and traceable: yes
- Control flow remains explicit: yes
- Shared authorization/business policy logic remains in engine flows: yes
- Permission decisions stay username-based: yes
- Per-connector message ordering guarantees preserved: yes
- Config precedence rules remain explicit and documented: yes

## Multi-Protocol / Connector

- Connector isolation maintained under multi-protocol runtime: yes
- Failure in one connector does not terminate others: yes
- Primary/default/secondary protocol semantics unchanged except documented config migration: yes

## Startup / Config / Compatibility

- Startup/load order verified against `aidocs/STARTUP_FLOW.md`: yes
- Config default/override behavior validated: yes
- Operator-visible changes documented in `UPGRADING-v3.md`: yes
- Compatibility note updated from planned -> implemented behavior: yes

## Tests

- Focused tests added/updated for provider-file loading: yes
- Relevant existing tests passing: yes
- Broader test pass status recorded: yes

## Documentation

- `aidocs/COMPONENT_MAP.md` updated: yes
- `aidocs/STARTUP_FLOW.md` updated: yes
- `conf/README.md` updated: yes
- `UPGRADING-v3.md` updated: yes

## Residual Risks / Follow-Ups

- `GetStartupMode`/`SetEnv` should remain rare in custom environment files; direct declarative environment defaults are preferred.
