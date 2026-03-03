# Impact Surface Report

## 1) Change Summary

- Slice family: `username-identity-contract-redesign`
- Goal:
  - move identity mapping contract ownership from engine `UserMap` to connectors
  - make canonical username the only engine security identity
  - remove engine outbound dependency on connector internal user IDs
  - move bot internal identity from single global state to protocol-scoped runtime state
  - define context-aware `GetBotAttribute("id")` resolution
- Out of scope:
  - redesigning group/help features
  - redesigning plugin/task extension API signatures
  - changing connector transport behavior beyond identity contract alignment

## 2) Subsystems Affected (with file anchors)

- Engine identity/config/runtime:
  - `bot/conf.go`
  - `bot/handler.go`
  - `bot/dispatch.go`
  - `bot/send_message.go`
  - `bot/pipecontext.go`
  - `bot/robot_connector_methods.go`
  - `bot/connector_runtime.go`
  - `bot/aidev_http.go`
  - `bot/robot.go`
  - `robot/connector_defs.go`
- Connectors:
  - `connectors/ssh/connector.go`
  - `connectors/ssh/server.go`
  - `connectors/slack/connectorMethods.go`
  - `connectors/slack/util.go`
  - `bot/term_connector.go`
  - `connectors/test/connectorMethods.go`
- Configuration:
  - `conf/protocols/ssh.yaml`
  - `conf/protocols/terminal.yaml`
  - `robot.skel/conf/protocols/ssh.yaml`
  - connector-specific protocol config docs/templates as needed
- Tests/docs:
  - `bot/*_test.go`
  - connector tests under `connectors/*`
  - `aidocs/STARTUP_FLOW.md`
  - `aidocs/SSH_CONNECTOR.md`
  - `aidocs/COMPONENT_MAP.md` (if connector responsibilities move materially)

## 3) Current Behavior Anchors

- Engine config still defines `UserMap map[string]string`: `bot/conf.go`.
- Runtime pushes per-protocol user maps to connectors via `SetUserMap(...)`: `bot/connector_runtime.go`.
- Inbound listed-user gate currently requires protocol mapping + directory membership: `bot/handler.go`.
- Outbound user sends attempt username->`<internalID>` resolution in engine: `bot/send_message.go`, `bot/pipecontext.go`.
- Slack currently treats engine-provided map as canonical override: `connectors/slack/util.go`.
- `SetBotID(...)` currently writes a single global bot ID: `bot/handler.go`.
- Only the primary connector can update bot identity state: `bot/connector_runtime.go` (`allowBotIdentity`).
- Thread-subscription self-check compares incoming `UserID` to one global bot ID: `bot/dispatch.go`.
- `GetBotAttribute("id")` currently returns that one global bot ID: `bot/robot.go`.

## 4) Proposed Behavior

- What changes:
  - engine no longer owns shared per-protocol `UserMap` contract
  - connectors parse and own protocol-local identity mapping config
  - engine policy identity uses trusted canonical username from connector
  - engine-to-connector user send path is username-based
  - `IgnoreUnlistedUsers` gates on username membership in global `UserRoster`
  - bot internal IDs are stored as protocol-scoped runtime state (`protocol -> botID`)
  - `GetBotAttribute("id")` resolves protocol-scoped bot ID by pipeline context:
    - inbound plugin/message pipeline protocol when present
    - `DefaultProtocol` for jobs/init/scheduled pipelines without inbound context
  - self-routing paths rely on connector `SelfMessage` contract rather than global bot-ID equality checks
  - user memory identity moves to username-based semantics
- What does not change:
  - startup phase ordering
  - connector isolation and runtime orchestration model
  - extension API method signatures

## 5) Invariant Impact Check

- Startup determinism preserved?: yes
- Explicit control flow preserved?: yes
- Shared auth/policy remains in engine flows?: yes
- Permission checks remain username-based?: yes (strengthened)
- Connector ordering guarantees preserved?: yes
- Config precedence still explicit?: yes (connector-local schemas documented per protocol)
- Multi-connector isolation preserved?: yes

Invariant text requiring updates:
- `aidocs/STARTUP_FLOW.md` identity mapping section currently references engine `UserMap`.

## 6) Cross-Cutting Concerns

- Startup sequencing impact:
  - connector config loading must remain deterministic while adding connector-local identity schemas.
- Config loading/merge/precedence impact:
  - engine removes shared identity map parsing; connector protocol configs become identity schema source.
- Execution ordering impact:
  - none expected if connector runtime orchestration remains unchanged.
- Resource lifecycle impact:
  - identity reload behavior must avoid race conditions and preserve connector isolation.
  - protocol-scoped bot identity updates must remain deterministic across startup/reload.

## 7) Concurrency Risks

- Shared state touched:
  - current user/channel maps in engine
  - protocol-scoped bot identity map in engine runtime
  - connector-local identity tables
  - runtime connector map/reload paths
- Risks:
  - stale identity tables during reload
  - stale/missing bot ID for a protocol during connector restart or delayed `SetBotID`
  - session/auth map races in SSH when keys update
- Mitigations:
  - lock/swap updates for identity tables
  - lock-protected protocol bot-ID map and deterministic fallback rules (`DefaultProtocol`)
  - deterministic reload ordering
  - focused race-sensitive tests for SSH and runtime reload

## 8) Backward Compatibility

- Existing robots/config expected impact:
  - configuration migration is expected and acceptable
  - connector protocol config templates must be updated in-repo
- Behavior changes for operators/users:
  - identity mapping config shifts to connector-specific fields
  - engine no longer requires global/per-protocol `UserMap`
- Migration/fallback plan:
  - staged slices with connector-by-connector migration
  - prioritize persistent brain compatibility
  - ephemeral memory format break acceptable

## 9) Validation Plan

- Focused tests:
  - inbound username trust + `IgnoreUnlistedUsers` behavior
  - outbound username-only send paths
  - `GetBotAttribute("id")` behavior by pipeline context (plugin vs job/init)
  - bot-self routing behavior without global bot-ID compare
  - connector-local identity schema parsing/validation (SSH/Slack/Terminal)
  - memory context key behavior by username
- Broader regression tests:
  - `go test ./bot`
  - `go test ./connectors/ssh`
  - `go test ./...`
  - `make test`
- Manual verification:
  - multi-protocol robot with same username across connectors
  - Slack canonical override collision scenario
  - SSH multi-key same-username login scenario

## 10) Documentation Plan

- Required updates during implementation:
  - `aidocs/STARTUP_FLOW.md` identity and `IgnoreUnlistedUsers` contract text
  - `aidocs/SSH_CONNECTOR.md` identity schema and guarantees
  - Slack/Terminal connector docs where identity contract is described
  - `devdocs/connector-identity-contract.md` if implementation details refine planned contract

## 11) Waiver (if applicable)

- Waived by: n/a
- Reason: n/a
- Scope limit: n/a
