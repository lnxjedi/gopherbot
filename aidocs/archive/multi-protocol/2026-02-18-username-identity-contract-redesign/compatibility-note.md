# Compatibility Note

## Change Summary

- Change:
  - Slice 1/2/3/3b/4/5/6/7 implementation:
    - outbound engine-to-connector user sends are now username-based
    - connector interface `SendProtocolUserChannelThreadMessage` no longer takes a separate internal `userid` parameter
    - inbound `IgnoreUnlistedUsers` now gates on connector-trusted username presence in global `UserRoster`
    - engine no longer parses/distributes `UserMap` identity mappings to connectors
    - connector identity mapping ownership moved to connector-local `ProtocolConfig`
    - bot internal IDs are protocol-scoped runtime state (`protocol -> botID`)
    - `GetBotAttribute("id")` resolves per pipeline context (inbound protocol vs `DefaultProtocol`)
    - SSH identity config is list-based (`ProtocolConfig.UserKeys` entries with `UserName` + `PublicKeys`)
    - test connector identity indexes are rebuilt per init/reload (no stale prior-user leakage)
    - ephemeral memory identity keys are username-based and thread keys are protocol-aware
    - removed global `currentCfg.botinfo.UserID` compatibility bridge; bot IDs are runtime protocol-scoped only
  - `UserID` remains inbound connector metadata/provenance.
- Why:
  - align runtime behavior with username-authoritative security model
  - reduce engine dependency on protocol-specific internal IDs
- Effective date/commit:
  - 2026-02-18 (working tree)

## What Stayed Compatible

- Unchanged behaviors:
  - extension API method signatures exposed to plugins/jobs/tasks remain unchanged
  - connector startup/runtime orchestration remains unchanged
- Unchanged config/env surfaces:
  - no required env var changes
  - existing connector protocol files still load when identity mapping is moved into connector `ProtocolConfig`

## What Changed

- Behavior differences:
  - outbound connector user targeting from engine now uses username identity only
  - `IgnoreUnlistedUsers` no longer requires protocol map membership; directory username membership is sufficient
  - `GetBotAttribute("id")` returns protocol-scoped bot IDs rather than a single global bot ID
  - `GetBotAttribute("id")` no longer falls back to legacy global config bot ID state
  - ephemeral memory recalls now use canonical usernames rather than connector `UserID`
  - thread-scoped ephemeral memory keys are separated by protocol + thread
- Startup/config/default differences:
  - no startup phase-order changes
  - top-level `UserMap` in `robot.yaml` or protocol files is invalid (config load fails)
- Identity/routing/connector differences:
  - connector implementations must resolve protocol-local user IDs from username internally on outbound sends
  - Slack canonical map now comes from `ProtocolConfig.UserMap`
  - SSH key mapping now comes from `ProtocolConfig.UserKeys` list entries (`UserName` + `PublicKeys`)
  - terminal/test connector local IDs come from connector `ProtocolConfig.Users` tables

## Operator Actions Required

- Required config changes:
  - move Slack canonical username->ID overrides to `ProtocolConfig.UserMap`
  - move SSH key mapping to `ProtocolConfig.UserKeys` list entries
  - remove legacy terminal/test `UserMap`/`AppendUserMap` usage; use `ProtocolConfig.Users`
  - for custom robots that should not inherit installed SSH defaults, set `ProtocolConfig.UserKeys: []` or provide explicit entries
- Optional config changes:
  - ensure connector-emitted usernames match canonical global `UserRoster` usernames
  - expect legacy persisted ephemeral memories keyed by old `UserID` semantics to be non-authoritative post-upgrade
- Environment variable changes:
  - none

## Rollout / Fallback

- Recommended rollout sequence:
  1. Deploy slice.
  2. Verify admin/security behaviors for known users with `IgnoreUnlistedUsers` enabled.
  3. Verify connector outbound user messaging across active protocols.
- Rollback/fallback instructions:
  - revert slice if connector user resolution regressions appear.
- Known temporary limitations:
  - none specific to identity keying; follow-on cleanup remains in final slice.

## Validation

- How to verify success:
  - run `go test ./bot ./connectors/ssh ./connectors/slack ./connectors/test`
  - run `go test ./...`
  - verify DM/reply/send behaviors in SSH and Slack by username.
- How to detect failure quickly:
  - user-targeted sends return `UserNotFound` for valid usernames
  - `IgnoreUnlistedUsers` blocks known `UserRoster` users despite correct connector username

## References

- Impact report: `aidocs/multi-protocol/2026-02-18-username-identity-contract-redesign/impact-surface-report.md`
- PR checklist: `aidocs/multi-protocol/2026-02-18-username-identity-contract-redesign/pr-invariants-checklist.md`
- Related docs:
  - `aidocs/STARTUP_FLOW.md`
  - `aidocs/SSH_CONNECTOR.md`
  - `devdocs/connector-identity-contract.md`
