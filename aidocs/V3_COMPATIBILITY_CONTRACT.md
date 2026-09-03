# v3 Compatibility Contract

## Pre-release development boundary

Until the first public v3 release, configuration, extension APIs, and
operational behavior are not frozen. They may change when needed to complete a
coherent, secure v3 design. Avoid needless churn and preserve useful
compatibility where it does not obstruct that design, but do not retain a
pre-release interface or behavior solely because a pilot Robot used it.

Pre-release changes must be explicit, bounded, and reviewable. In the same
logical change, keep current source, installed defaults, `robot.skel/`, tests,
user documentation, applicable decision records, migration guidance, and
material `CHANGELOG.md` entries aligned. Prefer a clear startup, validation, or
migration error over silently accepting obsolete configuration or behavior.

## Compatibility priorities during pre-release development

When choosing among otherwise valid designs, prioritize:

1. Preserve username-authoritative security behavior and the documented
   security invariants. These are architectural requirements, not migration
   conveniences.
2. Preserve persistent brain data where feasible. V2 import/export may remain
   an explicit CLI operation; normal v3 startup need not carry v2 branches.
3. Preserve extension behavior and signatures where doing so does not block
   the intended v3 model. When an extension API must change, fail clearly and
   document the required migration.

Configuration schema compatibility is not guaranteed. Any config migration
must update root `UPGRADING-v3.md`, installed defaults, the Robot skeleton,
tests, user documentation, and the relevant decision record in the same
change.

Installed extension defaults are the canonical baseline. Custom robots should
keep only enablement, credentials, parameters, and intentional local deltas.
Credentialed shipped extensions are opt-in, not active merely because they ship
with the engine.

## v3 release boundary

The first public v3 release establishes the stricter public compatibility
contract for the released configuration, extension APIs, and operational
behavior. Define and publish that contract as part of the release; do not infer
it from pre-release pilot behavior or this development policy.
