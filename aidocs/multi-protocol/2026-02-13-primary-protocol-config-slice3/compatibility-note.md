# Compatibility Note

## Change Summary

- Change:
  - Primary protocol config source is now explicit:
    - if `robot.yaml` contains `ProtocolConfig`, keep using it (compatibility mode + warning)
    - otherwise load required `conf/<PrimaryProtocol>.yaml`
  - primary protocol file can provide `UserMap` and `ChannelRoster`.
  - primary protocol-file `UserRoster` is treated as legacy ID mapping compatibility only (attributes ignored).
- Why:
  - reduce hidden include coupling and make primary protocol config loading deterministic.
- Effective date/commit:
  - 2026-02-13 (commit pending)

## What Stayed Compatible

- Unchanged behaviors:
  - robots embedding `ProtocolConfig` in `robot.yaml` continue to run.
  - secondary protocol loading behavior remains unchanged.
- Unchanged config/env surfaces:
  - `PrimaryProtocol`/`Protocol` key compatibility remains.

## What Changed

- Behavior differences:
  - missing `ProtocolConfig` in `robot.yaml` now triggers primary auto-load from `conf/<primary>.yaml`.
  - if primary file auto-load path is active and file/config is missing, config load fails.
- Startup/config/default differences:
  - explicit source precedence for primary protocol config.
- Identity/routing/connector differences:
  - none to runtime routing; identity map source may now include primary protocol file `UserMap`.

## Operator Actions Required

- Required config changes:
  - none if currently using embedded `ProtocolConfig` in `robot.yaml`.
- Optional config changes:
  - migrate primary protocol settings from `robot.yaml` includes into `conf/<primary>.yaml`.
- Environment variable changes:
  - none.

## Rollout / Fallback

- Recommended rollout sequence:
  1. Deploy slice.
  2. Watch for compatibility warnings about primary `ProtocolConfig` in `robot.yaml`.
  3. Migrate to preferred layout over time.
- Rollback/fallback instructions:
  - revert this slice if primary source-precedence behavior must be deferred.
- Known temporary limitations:
  - compatibility path still allows embedded protocol config in `robot.yaml` by design.

## Validation

- How to verify success:
  - with no primary `ProtocolConfig` in `robot.yaml`, confirm connector starts with config from `conf/<primary>.yaml`.
  - with embedded primary `ProtocolConfig`, confirm warning log and unchanged behavior.
- How to detect failure quickly:
  - startup/reload reports missing primary `conf/<primary>.yaml` or missing `ProtocolConfig` key.

## References

- Impact report: `aidocs/multi-protocol/2026-02-13-primary-protocol-config-slice3/impact-surface-report.md`
- PR checklist: `aidocs/multi-protocol/2026-02-13-primary-protocol-config-slice3/pr-invariants-checklist.md`
- Related docs:
  - `aidocs/STARTUP_FLOW.md`
