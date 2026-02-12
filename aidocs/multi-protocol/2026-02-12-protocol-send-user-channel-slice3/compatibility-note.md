# Compatibility Note

## Change Summary

- Change:
  - Added `SendProtocolUserChannelMessage(protocol, user, channel, message, ...)` to the Robot API.
  - Exposed the method in HTTP dispatch and shipped language libraries/bindings (JS, Lua, Bash, Python 2/3, Ruby, Julia, Yaegi wrapper).
  - Added focused bot test coverage for routing semantics.
- Why:
  - Enable explicit cross-protocol sends while preserving legacy send behavior and message routing invariants.
- Effective date/commit:
  - 2026-02-12 / this slice branch.

## What Stayed Compatible

- Unchanged behaviors:
  - Existing `SendUserMessage`, `SendUserChannelMessage`, `SendChannelMessage`, `Say`, `Reply`, and thread-based methods retain prior behavior.
  - Startup, reload, connector lifecycle, and config load order are unchanged.
- Unchanged config/env surfaces:
  - No config keys or environment variables changed.
  - Existing robot configs require no migration for this slice.

## What Changed

- Behavior differences:
  - New API allows explicit protocol target with semantics:
    - user + empty channel => DM
    - channel + empty user => channel message
    - user + channel => directed user-in-channel send
    - both empty => `MissingArguments`
- Startup/config/default differences:
  - none.
- Identity/routing/connector differences:
  - user lookup is protocol-aware for this method (protocol roster map first, then shared fallback map).
  - runtime rejects non-active target protocols with `Failed`.

## Operator Actions Required

- Required config changes:
  - none.
- Optional config changes:
  - none.
- Environment variable changes:
  - none.

## Rollout / Fallback

- Recommended rollout sequence:
  1. Pull updated engine.
  2. Use `SendProtocolUserChannelMessage` only where explicit cross-protocol targeting is needed.
- Rollback/fallback instructions:
  - revert plugins/jobs to existing `Send*` methods; no config rollback required.
- Known temporary limitations:
  - channel lookup remains global-map-based (protocol-scoped channel disambiguation is unchanged from prior behavior).

## Validation

- How to verify success:
  - call `SendProtocolUserChannelMessage("ssh", "alice", "", "test")` and verify DM on ssh.
  - call `SendProtocolUserChannelMessage("slack", "", "general", "test")` and verify channel message on slack.
  - call with both user+channel and verify directed message in channel.
- How to detect failure quickly:
  - unknown or inactive protocol returns `Failed` and logs protocol-not-active error.
  - empty protocol or empty user+channel returns `MissingArguments`.

## References

- Impact report:
  - `aidocs/multi-protocol/2026-02-12-protocol-send-user-channel-slice3/impact-surface-report.md`
- PR checklist:
  - `aidocs/multi-protocol/2026-02-12-protocol-send-user-channel-slice3/pr-invariants-checklist.md`
- Related docs:
  - `aidocs/EXTENSION_API.md`
