# AGENTS.md — Gopherbot

This file is the authoritative operating procedure for AI agents working in this repository.

If any ad-hoc instruction conflicts with this file, this file wins.

## Authority Boundary

- `AGENTS.md` is the single source of policy and required process.
- Skills may provide workflows/templates, but must not redefine repository policy.

## Phase 0 — Orientation (Two-Tier Model)

### Tier A: Default Orientation (required for all tasks)

Before proposing or implementing changes, read:
1. `aidocs/README.md`
2. `aidocs/COMPONENT_MAP.md`
3. root `GOALS_v3.md`

Then load only the canonical docs needed for the task scope.

### Tier B: Escalated Orientation (hard requirement when triggered)

You must run full architecture preflight before coding when **any** trigger applies.

Read in order:
1. `aidocs/README.md`
2. `aidocs/COMPONENT_MAP.md`
3. `aidocs/STARTUP_FLOW.md`
4. root `GOALS_v3.md`
5. `aidocs/TESTING_CURRENT.md`

Then summarize in your own words:
- core architectural invariants
- startup ordering constraints
- connector assumptions
- message routing model
- identity model

### Escalation Triggers (hard)

Escalated orientation is mandatory if a change touches or may affect:
- startup/config load order (`bot/start.go`, `bot/bot_process.go`, `bot/config_load.go`, `bot/conf.go`)
- message routing/pipeline ordering (`bot/dispatch.go`, `bot/run_pipelines.go`, scheduler flow)
- connector runtime/behavior (`connectors/*`, connector runtime orchestration)
- identity/authz semantics (username mapping, roster gates, authorization/elevation)
- root/default robot config structure (`conf/robot.yaml`, `robot.skel/conf/robot.yaml`)
- cross-protocol behavior/contracts
- privilege separation or task execution (`bot/privsep.go`, `bot/calltask.go`, `bot/task_execution.go`)
- user permission checks, admin/auth/elevation logic (`bot/available.go`, `bot/authorize.go`, `bot/elevate.go`)
- pre-pipeline user filtering or message context (`bot/handler.go`)

If uncertain, escalate.

## Phase 1 — Impact Analysis (Required for Cross-Cutting Changes)

For changes affecting connectors, routing, startup, configuration, identity, or compatibility:
- produce an Impact Surface Report before modifying code
- include subsystems, invariants, cross-cutting concerns, concurrency, compatibility, docs updates
- do not implement until report is shared, unless explicitly waived by user

## Architectural Invariants

Unless explicitly updated in canonical docs, these must hold:
- startup sequence is deterministic and traceable
- control flow is explicit, not implicit
- shared authorization/business policy remains in engine flows, not connectors
- permission/policy decisions are username-authoritative
- message routing order is preserved within a connector
- configuration precedence is explicit and documented
- engine-shipped extension defaults remain authoritative; custom robot extension config stays delta-only unless behavior is intentionally redefined
- multi-connector isolation prevents cascading failure
- secret access is explicit and scope-based: unprivileged extensions must not discover shared secrets through generic robot methods

## Connector Rules (Critical for Multi-Protocol)

- connectors own transport concerns and protocol-local behavior
- connectors must not bypass shared engine policy/authorization logic
- connectors map transport identity to canonical username deterministically
- cross-protocol identity equivalence is canonical username, not heuristic transport-ID matching
- connector failure isolation must be preserved when multiple connectors are enabled

## Extension Secret Boundary

- secrets may be exposed to an extension only through explicit administrator configuration for that extension, or through memory/brain state owned by that extension's authorized namespace
- unprivileged robot methods must not reveal shared secret-bearing configuration, nor provide indirect discovery of secrets outside the caller's granted scope
- do not add or document extension APIs that return provider registries, parameter-set contents, or other broad configuration objects containing secrets

## Security Model Invariants — Privilege Separation (setuid nobody)

These apply to `bot/privsep.go`, `bot/calltask.go`, `bot/task_execution.go`, `bot/run_pipelines.go`, and `bot/robot_pipecmd.go`. All are hard escalation triggers.

**Thread-pinning invariants — the most dangerous to break silently:**
- `dropThreadPriv` and `raiseThreadPrivExternal` both call `runtime.LockOSThread()` and intentionally **never** call `runtime.UnlockOSThread()`. This ensures the thread is destroyed rather than recycled. Never add `runtime.UnlockOSThread()` after these calls.
- `raiseThreadPrivExternal` permanently sets r/euid to `privUID/privUID`. That thread must never execute unprivileged work afterward.

**Pipeline privilege invariants:**
- `pipeContext.privileged` is set once at pipeline start from the starter task (`Plugin.Privileged` or `Job.Privileged`) in `startPipeline`. It must not be changed mid-pipeline.
- Adding privileged tasks/jobs/plugins to an unprivileged pipeline is blocked in `bot/robot_pipecmd.go`. This is a privilege-escalation gate — do not remove or bypass it.

**Child process boundary:**
- Interpreter-backed tasks (Lua/JS/Gsh/Yaegi Go) run in child RPC processes. The parent engine retains all policy, identity, and authorization authority. The child must never receive raw config objects, shared secret values, or privilege tokens it was not explicitly given through the task's configured parameters.

**Testing note:** `privSep` only activates on a setuid binary. There is no automated test for this path. Changes to `bot/privsep.go` or to privilege callsites in `bot/calltask.go` require manual testing: build the binary, `sudo chown nobody gopherbot && sudo chmod u+s gopherbot`, run as a non-root user, and verify `robot.log` contains `PRIVSEP - privilege separation initialized; daemon UID <uid>, unprivileged UID 65534`. Restore with `sudo chown $(whoami) gopherbot && sudo chmod u-s gopherbot` when done.

## Security Model Invariants — User Permission Model

These apply to `bot/handler.go`, `bot/available.go`, `bot/authorize.go`, `bot/elevate.go`, and `bot/run_pipelines.go`.

**Pre-pipeline filters — must remain first, before any worker or pipeline is created:**
- `IgnoreUsers` and `IgnoreUnlistedUsers` are checked in `handler.IncomingMessage` before any worker is created. They must remain pre-pipeline filters. Never move this logic into dispatch or pipeline code.
- The `IgnoreUsers` check is case-insensitive. New pre-pipeline user filtering must use the same comparison.

**Admin authority — sources are fixed:**
- Admin status (`isAdminUser` in `bot/available.go`) has exactly two legitimate sources: the `adminUsers` config list (username match), or `w.automaticTask == true`. It must never be derived from user input, message content, connector-provided flags, or any runtime state modifiable by users.
- `automaticTask == true` grants admin unconditionally. This is intentional: cron jobs are scheduled by administrators through robot configuration. If a future user-schedulable ("at-job") feature is added, it must **not** use `automaticTask = true` — it requires its own access control model.

**Check ordering in `run_pipelines.go` — must not be reordered:**
- Order is: **admin check → authorizer plugin → elevator plugin**. Admin check runs first because admins bypass the authorizer; elevation runs last because it is an additional confirmation step after base authorization is established.
- The `w.elevated` flag persists for the lifetime of the pipeline. Once elevated, subsequent tasks in the same pipeline do not re-challenge. Do not reset `w.elevated` mid-pipeline.

**Access control defaults:**
- `Task.Users` is a whitelist: an empty list means all users are permitted. Never invert this — empty must never restrict access.
- An authorizer plugin returning `robot.Normal` (0) is a mechanism failure, not success. Auth plugins must explicitly return `robot.Success` (1). Do not change this behavior.

## Security Model Invariants — Message Context and Privacy

The concern here is not command visibility (hard to hide) but message routing confidentiality: the bot accidentally broadcasting sensitive data to a channel, or treating a public channel message as if it were private.

**Connector authority over message context:**
- Connectors are the sole authority for `Incoming.DirectMessage`. This flag must be set accurately by the connector and must not be modified by the engine or plugins after `handler.IncomingMessage` returns.
- `DirectOnly: true` on a task is enforced in `pluginAvailable` before the pipeline starts — the task will not match in a channel. This enforcement must not be weakened.

**Response routing — no implicit privatization:**
- `r.Say()` and `r.Reply()` reply in the same channel/DM context as the triggering message. The engine does not implicitly privatize responses. This must not change.
- Plugins or tasks that return sensitive data (credentials, tokens, personal info, secrets) must either:
  - Be marked `DirectOnly: true` (command can only be invoked via DM), **or**
  - Explicitly call `r.Direct().Reply()` / `r.Direct().Say()` to force a DM response.
- Bot-initiated messages (not in response to a user command) containing per-user sensitive data must use `SendUserMessage` (DM path), not `SendChannelMessage`. There is no engine guard for this — it is a code review requirement.

## Documentation Discipline (Hard Mapping)

When behavior changes, update canonical docs in the same change:

- startup/config loading/order
  - `aidocs/STARTUP_FLOW.md`
- pipeline routing/execution ordering
  - `aidocs/PIPELINE_LIFECYCLE.md`
- scheduled job behavior
  - `aidocs/SCHEDULER_FLOW.md`
- connector behavior/identity mapping
  - connector-specific docs (`aidocs/SSH_CONNECTOR.md`, `aidocs/SLACK_CONNECTOR.md`, etc.)
  - `aidocs/COMPONENT_MAP.md` if component ownership/boundaries moved
- execution security / privilege separation behavior
  - `aidocs/EXECUTION_SECURITY_MODEL.md`
- extension API/runtime semantics
  - `aidocs/EXTENSION_API.md` and/or `aidocs/EXTENSION_SURFACES.md`
- compatibility/config migration behavior
  - `aidocs/V3_COMPATIBILITY_CONTRACT.md`
  - root `UPGRADING-v3.md`
  - corresponding defaults/templates in `conf/` and `robot.skel/`
- test harness assumptions
  - `aidocs/TESTING_CURRENT.md`

Additional rules:
- archive docs under `aidocs/archive/` are reference-only and non-canonical
- roadmap remains root `GOALS_v3.md` (human-maintained)
- AI "what's left" backlog remains `aidocs/TODO.md` (AI-maintained)

## Docs Hygiene Gate

For any change touching `aidocs/`, `devdocs/`, `AGENTS.md`, or `UPGRADING-v3.md`:
- run `helpers/check-docs-hygiene.sh`
- fix reported stale references/markers before completion

Stale documentation is a defect.

## v3 Compatibility Contract

Follow `aidocs/V3_COMPATIBILITY_CONTRACT.md`.

Required stance:
- extension API behavior compatibility is priority
- username-based security behavior is priority
- persistent brain compatibility is prioritized where feasible
- config backward compatibility is not required; migration is acceptable when documented

## Change Hygiene

- one logical change per branch
- planning before implementation for cross-cutting changes
- no silent refactors
- preserve behavior unless explicitly redefining it
- if redefining behavior, document migration strategy

## Post-Task Requirements

After implementation:
1. re-validate architectural invariants
2. update all required canonical docs
3. confirm startup sequence integrity
4. confirm connector isolation and ordering guarantees
5. run applicable tests and docs hygiene checks

For any task where integration tests are applicable:
6. run the applicable integration suite before closing out work — always redirect output to file, never stream to context:
   ```
   go test -run TestFoo -v --tags 'test integration netgo osusergo static_build' -mod readonly -race ./test \
     > /tmp/gopherbot-test.txt 2>&1; echo "EXIT:$?"
   ```
   Then read summary only: `grep -E "^(--- (PASS|FAIL)|FAIL\t|ok\t)" /tmp/gopherbot-test.txt`
   On failure, extract the failing test only: `awk '/=== RUN   TestFoo$/,/--- FAIL: TestFoo/' /tmp/gopherbot-test.txt`
7. classify every integration failure as either:
   - a real regression / newly introduced bug
   - an intentional behavior change with outdated test expectations
8. do not update test expectations until that classification is explicit
