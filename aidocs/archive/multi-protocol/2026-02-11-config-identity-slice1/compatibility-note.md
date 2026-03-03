# Compatibility Note (Draft)

Use this when behavior, config defaults, operator workflow, or externally visible semantics change.

## Change Summary

- Change: introduce `PrimaryProtocol` and `SecondaryProtocols`, keep `Protocol` as backward-compatible alias, and formalize username/roster identity rules for multi-protocol migration.
- Why: establish deterministic config and identity substrate before simultaneous multi-connector runtime.
- Effective date/commit: pending implementation

## What Stayed Compatible

- Unchanged behaviors:
  - existing robots using only `Protocol` continue to start.
  - single-connector runtime remains unchanged in this slice.
  - `IgnoreUnlistedUsers` behavior remains unchanged.
- Unchanged config/env surfaces:
  - existing environment-driven startup mode detection remains unchanged.
  - connector config files remain in robot `conf/` with template includes.

## What Changed

- Behavior differences:
  - if both `PrimaryProtocol` and `Protocol` exist and differ, `PrimaryProtocol` wins and warning is logged.
- Startup/config/default differences:
  - config schema introduces explicit primary/secondary protocol fields.
- Identity/routing/connector differences:
  - username mapping requirements are tightened (lowercase + digits policy; uppercase rejected).

## Operator Actions Required

- Required config changes:
  - none immediate (compatibility alias retained).
- Optional config changes:
  - migrate from `Protocol` to `PrimaryProtocol`.
  - add `SecondaryProtocols` when ready for later runtime slices.
- Environment variable changes:
  - none in this slice.

## Rollout / Fallback

- Recommended rollout sequence:
  1. deploy with compatibility alias support.
  2. migrate config repos to `PrimaryProtocol`.
  3. enable secondary protocols in later runtime slice.
- Rollback/fallback instructions:
  - remove `PrimaryProtocol`/`SecondaryProtocols` and continue using legacy `Protocol`.
- Known temporary limitations:
  - no simultaneous runtime connectors yet in this slice.

## Validation

- How to verify success:
  - robot starts with `Protocol` only.
  - robot starts with `PrimaryProtocol` only.
  - conflict case logs warning and chooses `PrimaryProtocol`.
- How to detect failure quickly:
  - startup fatal due to unknown protocol or malformed config.
  - user mapping failures from invalid username policy.

## References

- Impact report: `aidocs/multi-protocol/2026-02-11-config-identity-slice1/impact-surface-report.md`
- PR checklist: `aidocs/multi-protocol/2026-02-11-config-identity-slice1/pr-invariants-checklist.md`
- Related docs: `aidocs/STARTUP_FLOW.md`, `aidocs/README.md`, `bot/conf.go`
