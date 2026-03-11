# v3 Compatibility Contract

This document defines compatibility priorities for v3 work.

## Priority Guarantees

1. Extension runtime compatibility is required across v2 -> v3.
   - Existing plugin/job/task scripts should continue to run without API-signature churn.
   - Robot extension API method signatures and behavior are the compatibility boundary.
2. Username-based security behavior is required.
   - Admin users, groups, authorization, and policy checks stay username-authoritative.
3. Brain compatibility is prioritized.
   - Persistent brain data compatibility should be preserved whenever feasible.

## Explicit Non-Guarantee

- Configuration schema backward compatibility is not guaranteed for v3.
- Configuration migration is expected as architecture evolves.
- Failing fast on invalid/removed config keys is preferred over silently ignoring legacy keys.

## Configuration Layering Contract for Shipped Extensions

- Defaults shipped with the engine are the canonical baseline for included plugins/jobs/tasks.
- Custom robots should keep extension config override files minimal and local (enable/disable, parameters, environment-specific behavior).
- Avoid copying full shipped defaults into `custom/conf` unless intentionally redefining behavior.
- When behavior is intentionally redefined, document the divergence and keep only explicit delta in custom config.

## Required Contributor Actions for Config Changes

When a change requires config migration, contributors must:

1. Update root `UPGRADING-v3.md` in the same change.
2. Update default config files in `conf/` and robot templates in `robot.skel/`.
3. Update affected architecture docs in `aidocs/` and connector docs as needed.
4. Preserve the shipped-extension layering contract above (defaults authoritative; custom configs delta-only).

This keeps migration explicit while preserving runtime behavior for existing extension code.
