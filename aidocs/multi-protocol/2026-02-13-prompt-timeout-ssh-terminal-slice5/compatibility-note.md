# Compatibility Note

## Change Summary

- Change:
  - Prompt* timeout handling is now context-aware.
  - Shutdown now interrupts in-progress Prompt* waits immediately.
- Why:
  - Support longer interactive setup flows on local connectors (`ssh`, `terminal`) without forcing that behavior in team chat connectors.
  - Prevent long prompt waits from delaying stop/restart.
- Effective date/commit:
  - 2026-02-13 (commit pending)

## What Stayed Compatible

- Unchanged behaviors:
  - Default prompt timeout remains `45s` for non-ssh/terminal contexts.
  - Prompt waiter matching still keys by `user/channel/thread`.
  - Retry behavior for overlapping waiters remains unchanged.
- Unchanged config/env surfaces:
  - No YAML/env schema changes.

## What Changed

- Behavior differences:
  - Prompt timeout becomes `42m` for `ssh`/`terminal` when caller task is compiled Go or interpreter-backed (`.go`, `.lua`, `.js`).
  - Prompt waits return `Interrupted` immediately once shutdown starts.
- Startup/config/default differences:
  - Startup now resets a prompt-shutdown signal used to interrupt prompt waits.
- Identity/routing/connector differences:
  - none.

## Operator Actions Required

- Required config changes:
  - none.
- Optional config changes:
  - none.
- Environment variable changes:
  - none.

## Rollout / Fallback

- Recommended rollout sequence:
  1. Deploy normally.
  2. Verify prompt behavior in ssh/terminal and a non-interactive connector.
- Rollback/fallback instructions:
  - Revert this slice if prompt semantics are not desired.
- Known temporary limitations:
  - Extended timeout classification currently targets compiled Go and `.go/.lua/.js` interpreter-backed tasks.

## Validation

- How to verify success:
  - In ssh/terminal, invoke a plugin that prompts and confirm prompt stays active beyond `45s`.
  - Trigger shutdown during a pending prompt and confirm pipeline prompt returns quickly.
- How to detect failure quickly:
  - Prompt still timing out at `45s` in ssh/terminal interpreted flows.
  - Stop/restart waiting for long prompt timeout windows.

## References

- Impact report: `aidocs/multi-protocol/2026-02-13-prompt-timeout-ssh-terminal-slice5/impact-surface-report.md`
- PR checklist: `aidocs/multi-protocol/2026-02-13-prompt-timeout-ssh-terminal-slice5/pr-invariants-checklist.md`
- Related docs:
  - `aidocs/PIPELINE_LIFECYCLE.md`
  - `aidocs/EXTENSION_API.md`
  - `aidocs/SSH_CONNECTOR.md`
