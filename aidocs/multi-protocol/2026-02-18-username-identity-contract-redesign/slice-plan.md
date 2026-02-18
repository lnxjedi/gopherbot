# Slice Plan: Username Identity Contract Redesign

This plan sequences implementation in thin, testable slices.

## Slice 1: Contract Scaffolding in Engine Interfaces

Goal:
- introduce username-authoritative send contract while preserving temporary compatibility paths.

Scope:
- `robot/connector_defs.go`
- engine wrapper calls in `bot/send_message.go`, `bot/connector_runtime.go`
- connector compile updates with compatibility shims

Expected result:
- engine can target connector users by username without requiring engine-side `UserID` resolution.

Tests:
- runtime connector tests around send routing and API behavior.

Docs:
- compatibility note for connector interface transition.

## Slice 2: Inbound Identity and IgnoreUnlistedUsers

Goal:
- make connector-provided canonical username authoritative in engine policy.

Scope:
- `bot/handler.go`
- remove hard dependency on engine protocol ID map for listed-user gating
- keep `UserID` as metadata only

Expected result:
- `IgnoreUnlistedUsers` uses trusted username + global `UserRoster`.

Tests:
- `handler` identity tests for mapped/unmapped behavior
- ignore-unlisted coverage for username-only gating.

Docs:
- update identity sections in `aidocs/STARTUP_FLOW.md`.

## Slice 3: Remove Engine UserMap from Config/Runtime

Goal:
- stop parsing and distributing engine `UserMap`.

Scope:
- `bot/conf.go`
- `bot/connector_runtime.go`
- remove/replace `SetUserMap` contract surface

Expected result:
- no engine-owned per-protocol identity map contract.

Tests:
- config load tests
- runtime connector initialization/reload tests.

Docs:
- startup/config precedence updates in `aidocs/STARTUP_FLOW.md`.

## Slice 3b: Protocol-Scoped Bot Identity in Engine

Goal:
- replace single global bot internal ID with protocol-scoped bot ID runtime state.
- define deterministic `GetBotAttribute("id")` behavior in multi-protocol pipelines.

Scope:
- `bot/handler.go` (`SetBotID` protocol-scoped recording path)
- `bot/connector_runtime.go` (protocol-aware identity updates from connector handlers)
- `bot/robot.go` (`GetBotAttribute("id")` resolution by pipeline protocol/default protocol)
- `bot/dispatch.go` (remove global bot-ID comparisons; rely on `SelfMessage` semantics)
- any startup/runtime state structs that currently hold one bot ID (`currentCfg.botinfo.UserID` compatibility bridge)

Expected result:
- connectors still call `SetBotID`, but engine stores `protocol -> botID`.
- plugin/message pipelines get the triggering protocol bot ID from `GetBotAttribute("id")`.
- job/init/scheduled pipelines get the default protocol bot ID from `GetBotAttribute("id")`.
- no behavior depends on a single global bot internal ID.

Tests:
- `GetBotAttribute("id")` in plugin pipelines returns per-protocol ID.
- `GetBotAttribute("id")` in scheduled/init/job pipelines resolves to `DefaultProtocol` ID.
- thread-subscription/self-routing behavior is preserved without global bot-ID compare.
- integration harness configs under `test/*/conf/robot.yaml` are updated to keep canonical bot identity in robot config (not connector-local), with matching protocol test config updates as needed.

Docs:
- update identity sections in `aidocs/STARTUP_FLOW.md`.
- update `devdocs/connector-identity-contract.md` bot ID semantics.

## Slice 4: SSH Connector Local Identity Schema

Goal:
- move SSH identity mapping fully into SSH protocol config.

Scope:
- `connectors/ssh/*`
- `conf/protocols/ssh.yaml`
- `robot.skel/conf/protocols/ssh.yaml`

Expected result:
- SSH supports `username -> [pubkeys...]` and canonical username trust boundary.

Tests:
- auth lookup tests for multi-key usernames
- reload/update behavior tests.
- integration config fixtures updated where SSH identity/schema fields changed.

Docs:
- `aidocs/SSH_CONNECTOR.md` identity section updates.

## Slice 5: Slack/Terminal/Test Connector Alignment

Goal:
- align remaining connectors to connector-local identity config ownership.

Scope:
- `connectors/slack/*`
- `bot/term_connector.go`
- `connectors/test/*`
- relevant default config files

Expected result:
- Slack canonical override behavior preserved via connector-local config.
- Terminal/test remain username-authoritative with local protocol IDs.

Tests:
- Slack user lookup/canonical precedence tests
- terminal/test send/lookup behavior tests.
- integration config fixtures under `test/*/conf/` updated to reflect connector-local vs robot-global key ownership.

Docs:
- connector-specific docs for identity config schemas.

## Slice 6: Memory Identity Migration

Goal:
- key user-scoped memory by username semantics.

Scope:
- `bot/robot.go`
- `bot/ephemeral.go`
- any read/write paths depending on old key shape

Expected result:
- cross-protocol same-username memory behavior is consistent.
- ephemeral memory compatibility break accepted.

Tests:
- memory context tests for username semantics
- thread-context behavior tests.
- integration tests and fixtures updated where expected behavior or keying assumptions change.

Docs:
- note compatibility expectations (ephemeral vs persistent brain priorities).

## Slice 7: Cleanup and Contract Finalization

Goal:
- remove temporary compatibility shims and old map code paths.

Scope:
- remove dead code in engine/connectors
- finalize docs and invariants language

Expected result:
- single coherent username-authoritative contract across code and docs.

Tests:
- full regression pass.

Docs:
- final pass on `aidocs/STARTUP_FLOW.md`, connector docs, and related multi-protocol artifacts.
