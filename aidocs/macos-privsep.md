# macOS Privilege Separation Design

Status: planned design. macOS native builds are supported, but macOS privilege separation is not currently available.

## Problem

The current privilege-separation implementation in `bot/privsep.go` is compiled only on Linux and BSD platforms. It uses `setreuid` plus `runtime.LockOSThread()` to make UID changes affect one locked OS thread at a time.

That model is intentionally delicate:

- `dropThreadPriv` and `raiseThreadPrivExternal` must never unlock their OS thread.
- A permanently raised external-task thread must never execute unprivileged work afterward.
- Correctness depends on Go runtime thread lifetime behavior, not on an explicit process boundary.

Darwin has enough UID syscall surface to make parts of this compile, but the current implementation does not compile as-is because `golang.org/x/sys/unix.Gettid` is not exposed on Darwin. More importantly, compiling the old model on macOS would not prove that thread-scoped credential changes are safe under the Go runtime and Darwin process model.

## Goal

Move privilege separation away from per-thread UID transitions and toward an explicit process model that can later support macOS without relying on thread credential behavior.

The desired end state:

- The parent engine remains the only policy authority.
- Privilege level is selected before external/interpreted work crosses a process boundary.
- Broker processes permanently commit to one UID and never switch back.
- Normal task execution does not call `setreuid` inside a multithreaded Go runtime.
- Linux/BSD behavior remains secure during migration.

## Non-Goals

- Do not make connectors responsible for authorization or privilege decisions.
- Do not move policy, parameter resolution, or secret selection into child processes.
- Do not treat macOS setuid behavior as equivalent to Linux/BSD without manual proof.
- Do not weaken the existing rule that unprivileged extensions cannot discover broad secret-bearing configuration.

## Proposed Model

Start permanent execution brokers early in startup:

- privileged broker: permanently runs as the invoking robot user
- unprivileged broker: permanently runs as the configured unprivileged account, usually `nobody`

Each broker is a child `gopherbot` process entered through an internal command such as `privsep-broker`. The broker receives only a narrow RPC surface for launching already-authorized external/interpreted work.

The parent engine keeps responsibility for:

- message routing
- admin checks
- authorizer and elevator execution order
- pipeline privilege classification
- extension parameter assembly
- secret scoping
- cancellation and timeout policy
- operator-visible pipeline state

The broker handles only:

- verifying its process UID at startup
- launching child execution with inherited broker privilege
- returning stdout, stderr, exit status, and lifecycle events to the parent
- terminating broker-owned child process groups when requested

## Startup Sequence

Broker startup must happen before normal workload execution and before untrusted extensions can run.

Proposed sequence:

1. Detect whether setuid privilege separation is active.
2. Start the privileged broker and unprivileged broker while both identities are still available.
3. Each broker permanently sets real and effective UID to its assigned identity.
4. Each broker reports its role, real UID, effective UID, pid, and protocol version.
5. Parent validates the broker report before enabling broker-backed execution.
6. If a required broker is unavailable, parent refuses external work that requires that privilege class and logs a startup error.

This preserves explicit startup control flow and makes broker availability traceable.

## Execution Routing

Current task policy should remain parent-owned:

- `startPipeline` sets pipeline privilege from the starter plugin/job.
- `bot/robot_pipecmd.go` blocks privileged tasks from being added to unprivileged pipelines.
- `runPipeline` and `executeTask` classify task type and execution path.

Under the broker model:

- compiled-in Go plugins/jobs/tasks remain in-process trusted code
- external executable tasks run through the broker for the pipeline privilege
- Lua, JavaScript, Gopherbot shell, and interpreted Go RPC child execution run through the broker for the pipeline privilege
- default-config loading for external plugins runs through the unprivileged broker unless an explicit privileged path is required and already authorized

The parent should choose the broker from engine policy. The child must not decide whether it is allowed to run privileged.

## Security Invariants

- Brokers must permanently commit to one UID. No broker process may alternate between privileged and unprivileged work.
- A broker must verify and report its real/effective UID before accepting launch requests.
- Parent must reject broker requests if role, UID, or protocol version does not match expected startup state.
- Parent must pass only task-specific environment, argv, working directory, and already-selected parameters.
- Parent must not pass raw robot config, provider registries, broad parameter sets, or privilege tokens to brokers.
- Broker child processes should run in separate process groups so timeout and admin kill behavior remains scoped.
- Broker failure must fail only that privilege class and must not cascade across connectors or unrelated pipelines.

## macOS Considerations

macOS currently uses `bot/privsep_unsupported.go`, where `privSep` remains false and privilege helpers are no-ops.

Before enabling macOS privilege separation:

- verify setuid execution behavior on current macOS
- verify whether a setuid-to-nobody binary can preserve both required identities long enough to spawn permanent brokers
- verify broker UID commitment through manual tests, not just compilation
- verify child process group kill behavior
- verify code signing, quarantine, and filesystem ownership interactions for setuid binaries

If macOS cannot safely support the inverted setuid-nobody model, keep macOS as a native development/runtime platform without privilege separation.

## Migration Plan

1. Introduce broker process entrypoint behind a feature flag on Linux/BSD.
2. Route one external execution path through brokers while preserving current thread privsep as fallback.
3. Extend broker routing to interpreter-backed RPC paths.
4. Remove normal task execution calls to `dropThreadPriv`, `raiseThreadPriv`, and `raiseThreadPrivExternal`.
5. Retain minimal startup-only privilege setup code for broker creation.
6. Add macOS manual validation only after Linux/BSD broker behavior is stable.

## Validation Plan

Automated tests:

- unit tests for broker role/UID validation
- task-routing tests confirming privileged and unprivileged pipelines select the expected broker
- timeout/kill tests confirming process group cleanup
- regression tests for parameter and secret scoping

Manual setuid tests:

- build the binary
- set owner/setuid according to the target platform procedure
- run as a non-root robot user
- verify startup logs show expected parent and broker UIDs
- run privileged and unprivileged probe extensions
- verify unprivileged probes cannot access privileged-only files or secrets
- restore binary ownership and setuid bits after testing

## Documentation Updates Required With Implementation

- `aidocs/EXECUTION_SECURITY_MODEL.md`
- `aidocs/PIPELINE_LIFECYCLE.md`
- `aidocs/STARTUP_FLOW.md`
- `aidocs/TESTING_CURRENT.md`
- root `GOALS_v3.md`
