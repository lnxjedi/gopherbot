# v3 Compatibility Contract

Compatibility priorities, in order:

1. Preserve extension API behavior and signatures for existing
   plugins/jobs/tasks. A security-obsolete API may be removed only if it fails
   clearly and the migration is documented.
2. Preserve username-authoritative security behavior.
3. Preserve persistent brain data where feasible. V2 import/export may remain
   an explicit CLI operation; normal v3 startup need not carry v2 branches.

Configuration schema compatibility is not guaranteed. Prefer a fail-fast
migration error over silently accepting a removed key or template function.
Any config migration must update root `UPGRADING-v3.md`, installed defaults,
the robot skeleton, and the relevant decision record in the same change.

Installed extension defaults are the canonical baseline. Custom robots should
keep only enablement, credentials, parameters, and intentional local deltas.
Credentialed shipped extensions are opt-in, not active merely because they ship
with the engine.
