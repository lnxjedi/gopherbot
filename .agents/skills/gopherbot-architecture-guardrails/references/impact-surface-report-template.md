# Impact Surface Report Template

Use this before non-trivial changes (unless explicitly waived by the user).

## 1) Change Summary

- Slice name:
- Goal:
- Out of scope:

## 2) Subsystems Affected (with file anchors)

- Files/directories expected to change:
- Key functions/types/symbols:

## 3) Current Behavior Anchors

- Startup/order anchors:
- Routing/message-flow anchors:
- Identity/authorization anchors:
- Connector behavior anchors:

## 4) Proposed Behavior

- What changes:
- What does not change:

## 5) Invariant Impact Check

- Startup determinism preserved?:
- Explicit control flow preserved?:
- Shared auth/policy remains in engine flows?:
- Permission checks remain username-based?:
- Connector ordering guarantees preserved?:
- Config precedence still explicit?:
- Multi-connector isolation preserved (if applicable)?:

If any invariant is redefined, explain why and list required doc updates.

## 6) Cross-Cutting Concerns

- Startup sequencing impact:
- Config loading/merge/precedence impact:
- Execution ordering impact:
- Resource lifecycle impact (connections, goroutines, shutdown):

## 7) Concurrency Risks

- Shared state touched:
- Locking/channel/event-order assumptions:
- Race/deadlock/starvation risks:
- Mitigations:

## 8) Backward Compatibility

- Existing robots/config expected impact:
- Behavior changes for operators/users:
- Migration/fallback plan:

## 9) Validation Plan

- Focused tests:
- Broader regression tests:
- Manual verification steps:

## 10) Documentation Plan

- `aidocs/STARTUP_FLOW.md` updates:
- `aidocs/COMPONENT_MAP.md` updates:
- Connector doc updates:
- Other docs:

## 11) Waiver (if applicable)

- Waived by:
- Reason:
- Scope limit:

