# macOS Privilege Separation Plan

Status: implementation started. macOS native builds are supported, and a Darwin child-role implementation exists, but manual setuid validation is still required before macOS privilege separation is considered production-ready.

## Problem

The legacy privilege-separation implementation in `bot/privsep.go` was compiled only on Linux and BSD platforms. It used `setreuid` plus `runtime.LockOSThread()` to make UID changes affect one locked OS thread at a time.

That model is intentionally delicate:

- `dropThreadPriv` and `raiseThreadPrivExternal` must never unlock their OS thread.
- A permanently raised external-task thread must never execute unprivileged work afterward.
- Correctness depends on Go runtime thread lifetime behavior, not on an explicit process boundary.

Darwin has enough UID syscall surface to make parts of this compile, but extending the thread-scoped model to macOS would not prove that credential changes remain safe under the Go runtime and Darwin process model.

## Goal

Move privilege separation away from per-thread UID transitions and toward a one-shot process model for file-backed extensions.

The desired end state:

- The parent engine remains the only policy authority.
- Every extension that is not compiled into the engine crosses a child-process boundary before execution.
- The parent selects the child privilege class before the extension crosses that boundary.
- The child permanently commits to either the invoking robot user or the unprivileged account before starting any interpreter or external command.
- Startup fails closed when privilege separation is active and retained supplementary groups do not match explicit administrator policy.
- Normal task execution does not call `setreuid` inside the multithreaded parent process.
- Linux/BSD and macOS may use different low-level credential setup, but expose the same parent/child execution contract.

## Non-Goals

- Do not introduce long-lived broker processes as the base design.
- Do not make connectors responsible for authorization or privilege decisions.
- Do not move policy, parameter resolution, or secret selection into child processes.
- Do not support running compiled-in Go extensions as unprivileged code. Compiled-in extensions remain trusted engine code.
- Do not treat macOS setuid behavior as equivalent to Linux/BSD without manual proof.
- Do not weaken the existing rule that unprivileged extensions cannot discover broad secret-bearing configuration.

## Proposed Model

Use one child process per file-backed extension invocation.

The parent engine keeps responsibility for:

- message routing
- admin checks
- authorizer and elevator execution order
- pipeline privilege classification
- extension parameter assembly
- secret scoping
- cancellation and timeout policy
- operator-visible pipeline state

The child process handles only:

- committing permanently to the parent-selected privilege class
- verifying its real/effective UID and GID before extension execution
- starting the built-in interpreter for Yaegi/Go, JavaScript, Lua, or Gopherbot shell
- or execing the external interpreter/script path for Ruby, Python, Bash, and other executable extensions
- returning stdout, stderr, exit status, and RPC responses to the parent

The child must not decide whether privileged execution is allowed. It receives only the parent-selected execution role and task-specific execution context.

## Execution Routing

Current parent-owned task policy remains authoritative:

- `startPipeline` sets pipeline privilege from the starter plugin/job.
- `bot/robot_pipecmd.go` blocks privileged tasks from being added to unprivileged pipelines.
- `runPipeline` and `executeTask` classify task type and execution path.

Under the process-oriented model:

- compiled-in Go plugins/jobs/tasks remain in-process trusted code
- external executable tasks run in a one-shot child for the selected privilege class
- Lua, JavaScript, Gopherbot shell, and interpreted Go run in a one-shot child before their built-in interpreter starts
- external Ruby, Python, Bash, and similar scripts run in a one-shot child before the external interpreter or script is execed
- external plugin default-config retrieval uses the same child boundary, with conservative unprivileged execution unless the extension is explicitly configured as privileged and the parent has already selected the privileged role

The current `pipeline-child-exec` and `pipeline-child-rpc` paths are the natural implementation targets. A shared child credential preamble should run before either child path starts interpreter/runtime work.

## macOS Proof Of Concept

The local proof of concept in `../go-test` demonstrates the core Darwin credential sequence for a binary owned by `nobody:nobody` with setuid and setgid bits set.

At initial exec:

- `RUID` is the invoking user
- `EUID` is `nobody`
- saved UID is `nobody`
- `RGID` is the invoking primary group
- `EGID` is `nobody`

The parent swaps only effective credentials back to the invoking user:

- `setreuid(-1, ruid)`
- `setregid(-1, rgid)`

On Darwin this leaves the saved UID/GID as `nobody`, giving the parent enough state to re-exec the setuid binary for children without ever using root.

Each child re-execs the same setuid binary so the kernel restores effective/saved credentials to `nobody`, then permanently commits:

- invoking-user child:
  - `setregid(rgid, rgid)`
  - `setreuid(ruid, ruid)`
- nobody child:
  - `setegid(nobodyGID)`
  - `seteuid(nobodyUID)`
  - `setregid(nobodyGID, nobodyGID)`
  - `setreuid(nobodyUID, nobodyUID)`

The reported proof-of-concept run showed:

- parent initial `RUID=502`, `EUID=-2`
- parent after swap `EUID=502`
- invoking-user child `RUID=502`, `EUID=502`, `RGID=20`, `EGID=20`
- unprivileged child `RUID=-2`, `EUID=-2`, `RGID=-2`, `EGID=-2`

This validates the UID and primary-GID direction for macOS, but it does not complete the production security model by itself.

## Open Security Issue: Supplementary Groups

The proof-of-concept unprivileged child retained the invoking user's supplementary groups.

That means the child was not fully equivalent to a fresh `nobody:nobody` login context. It had nobody as real/effective UID and primary GID, but it could still inherit access through group-readable filesystem permissions.

Darwin does not provide a practical non-root way for this inverted setuid-nobody model to drop those supplementary groups after exec. The production model therefore needs an explicit administrator policy for retained groups.

Implementation must resolve this before claiming acceptable unprivileged isolation:

- clear or replace supplementary groups before extension execution on platforms where the platform allows it
- or fail startup unless retained supplementary groups are explicitly allowed by configuration
- or explicitly reject privilege separation on that platform/configuration

Until this is implemented, the macOS proof of concept should be treated as proof that process-oriented UID commitment can work, not proof that the full unprivileged execution boundary is complete.

## Configuration

Root `robot.yaml` privilege-separation controls:

```yaml
PrivsepAllowAllSupplementaryGroups: false
PrivsepAllowedSupplementaryGroups: []
```

Semantics:

- `PrivsepAllowAllSupplementaryGroups` defaults to `false`.
- `PrivsepAllowedSupplementaryGroups` is a list of numeric group IDs that may remain in the unprivileged child supplementary group set.
- When privilege separation is active, startup must run a self-check that observes the effective unprivileged child UID, primary GID, and supplementary groups.
- Startup must fail closed if any retained supplementary group is not explicitly handled by policy.
- If `PrivsepAllowAllSupplementaryGroups: true`, startup may continue with all retained groups, but must log a high-severity warning/audit line because this weakens the unprivileged boundary.
- If `PrivsepAllowAllSupplementaryGroups: false`, startup may continue only when every retained supplementary group is either platform-required for the unprivileged identity or listed in `PrivsepAllowedSupplementaryGroups`.
- Config parsing should reject negative group IDs in `PrivsepAllowedSupplementaryGroups`; use the unsigned/integer value reported by the platform probe for groups such as macOS `nobody`.

Installed `conf/robot.yaml` sets the strict defaults above. `robot.skel/conf/robot.yaml` includes commented examples for allowing all retained groups or listing specific numeric group IDs. A robot administrator should have to opt in before unprivileged extensions retain any group-derived authority.

## Operator Privilege Guidance

Grant privileged robot capabilities to the invoking robot user, not to a group that an unprivileged child might retain.

Recommended patterns:

- grant narrowly scoped sudoers entries directly to the robot username
- grant file ownership or ACL access directly to the robot user where possible
- keep secret files unreadable by broad operator groups
- keep the robot invoking user out of broad administrative groups such as `wheel` unless the deployment intentionally accepts that those group privileges may be visible to unprivileged children on platforms that cannot drop supplementary groups

Avoid patterns:

- granting robot privilege through `%wheel`, `%admin`, or similar broad sudoers groups
- making cloud credential files or deployment keys group-readable by a group retained by unprivileged children
- relying on primary UID/GID changes alone to protect resources that are also readable through supplementary groups

The practical rule is: if an unprivileged child may retain supplementary group membership, group grants are not a reliable separation boundary. Use direct user grants for privileged robot work.

## Linux EC2 Metadata Firewall Note

On Linux hosts running in AWS EC2, instance metadata endpoints can expose temporary credentials for the instance role. AWS documents IMDS endpoints at IPv4 `169.254.169.254` and, when IPv6 IMDS is explicitly enabled on Nitro instances, `[fd00:ec2::254]` (see AWS EC2 [Configure the Instance Metadata Service options](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/configuring-instance-metadata-options.html)).

Deployments that rely on privilege separation should consider UID-scoped firewall rules so unprivileged extension children cannot reach IMDS, while the privileged robot user can if the robot legitimately needs instance role credentials.

Implementation guidance for operators:

- prefer IMDSv2 and least-privilege instance profiles
- block metadata endpoint access for the unprivileged UID with Linux owner-match firewall rules (`iptables`/`nftables`, depending on host policy)
- cover both IPv4 and IPv6 IMDS endpoints when IPv6 metadata is enabled
- treat metadata access as credential access, not ordinary network access

This is an operational hardening recommendation, not a substitute for the engine's secret-scoping rules.

## Platform Mechanics

### macOS

Expected operating model:

1. Install the binary owned by `nobody:nobody` with setuid and setgid bits.
2. At startup, detect the inverted setuid-nobody state.
3. Move the parent engine back to the invoking user while preserving the ability to re-exec children through the setuid binary.
4. Run a startup self-check for unprivileged child UID/GID and supplementary group policy.
5. For each file-backed extension invocation, start a child by re-execing the same binary with an internal child command.
6. In the child command, permanently commit to the requested role before starting any interpreter, RPC loop, or external executable.
7. Verify real/effective UID and GID, and fail closed if they do not match the requested role.

macOS-specific validation still required:

- supplementary group handling
- child process group kill behavior
- code signing, quarantine, and filesystem ownership interactions for setuid binaries
- behavior when `nobody` has UID/GID `-2` as exposed through Go/syscall APIs

### Linux/BSD

Linux/BSD now use the same one-shot child role contract for file-backed extension execution. Limited thread-scoped helpers remain only for parent-owned operations and migration compatibility.

The target process contract should still be the same:

- parent chooses role
- child commits permanently before extension execution
- no file-backed extension runs directly in the multithreaded parent
- thread-scoped UID switching is removed from normal extension execution once process children cover all external/interpreter paths

The low-level Linux/BSD credential sequence may differ from macOS and should be validated independently.

## Security Invariants

- One child invocation has exactly one privilege class.
- No child process may alternate between privileged and unprivileged work.
- A child must verify and report, or at minimum fail closed on, real/effective UID and GID mismatch before executing extension code.
- Supplementary groups must be explicitly handled before the unprivileged role is considered complete.
- Privsep startup must fail closed when retained supplementary groups are outside administrator policy.
- Parent must pass only task-specific environment, argv, working directory, and already-selected parameters.
- Parent must not pass raw robot config, provider registries, broad parameter sets, or privilege tokens to children.
- Parent-owned RPC remains the only path for Robot API calls from interpreter-backed children.
- Child processes should run in separate process groups so timeout and admin kill behavior remains scoped.
- Child failure must fail only that invocation and must not cascade across connectors or unrelated pipelines.

## Migration Plan

1. Add a small shared child credential preamble that can commit a just-execed child to either invoking-user or unprivileged role. (Implemented)
2. Keep limited thread-scoped helpers only for parent-owned operations and migration compatibility. (Implemented)
3. Route external executable execution through the credential preamble before `pipeline-child-exec` runs the target command. (Implemented)
4. Route interpreter-backed RPC execution through the credential preamble before `pipeline-child-rpc` starts Yaegi/Go, JavaScript, Lua, or Gopherbot shell runtime work. (Implemented)
5. Route external plugin default-config retrieval through the same child boundary. (Implemented)
6. Remove normal task execution calls to `dropThreadPriv`, `raiseThreadPriv`, and `raiseThreadPrivExternal` after all file-backed execution paths use committed child processes. (Implemented for file-backed extension execution)
7. Retain only minimal startup/child-creation privilege setup code.
8. Add `robot.yaml` supplementary-group policy parsing and startup self-check failure behavior. (Implemented)
9. Enable macOS only after manual validation covers UID, primary GID, supplementary groups, process-group cleanup, and setuid binary lifecycle.

## Validation Plan

Automated tests:

- unit tests for role parsing and UID/GID validation logic
- unit tests for supplementary-group policy parsing and fail-closed startup decisions
- task-routing tests confirming privileged and unprivileged pipelines select the expected child role
- timeout/kill tests confirming process group cleanup
- regression tests for parameter and secret scoping

Manual setuid tests:

- build the binary
- set owner/setuid/setgid according to the target platform procedure
- run as a non-root robot user
- verify startup logs show expected parent and child UIDs/GIDs
- verify supplementary groups for unprivileged children are cleared or otherwise match the accepted security model
- verify startup fails when retained supplementary groups are present and not allowed by policy
- run privileged and unprivileged probe extensions
- verify unprivileged probes cannot access privileged-only files or secrets
- on Linux EC2 deployments, verify UID-scoped firewall rules block IMDS access from the unprivileged UID
- restore binary ownership and setuid bits after testing

## Documentation Updates Required With Implementation

- `aidocs/EXECUTION_SECURITY_MODEL.md`
- `aidocs/PIPELINE_LIFECYCLE.md`
- `aidocs/STARTUP_FLOW.md`
- `aidocs/TESTING_CURRENT.md`
- root `GOALS_v3.md`
