# Execution and Security Decisions

The parent engine is the sole policy authority. Connectors and child runtimes
provide facts or execute already-authorized work; they never receive authority
tokens that let them make policy decisions.

## User and message trust

- Canonical username is authoritative for admin, authorization, elevation,
  groups, and memory ownership.
- A connector assertion is trusted for policy only with `ValidatedUser=true`.
- `IgnoreUsers` and `IgnoreUnlistedUsers` run before a worker or pipeline
  exists. The sole exception is the narrow validation-code path that lets an
  unvalidated transport account prove its ID to an administrator.
- Admin status comes only from configured usernames or `automaticTask`.
- An empty `Task.Users` whitelist permits all users.
- Security order is admin, private-context rules, authorizer, then elevator.
  Auth/elevator plugins must return `robot.Success`; zero/`Normal` is failure.

Elevation proves current control of an identity after authorization; it does
not grant permission. Once a pipeline elevates, it remains elevated for that
pipeline only.

## Message confidentiality

Connectors alone define whether input was a DM, hidden invocation, or self
message. The engine enforces private-command requirements before extension
code.

`Say()` and `Reply()` intentionally preserve the triggering route. They do not
silently move sensitive output to a DM. Sensitive extensions must require a
private command or call `Direct()`. Proactive per-user sensitive messages must
use the user-message/DM path.

Channel restrictions are normally visibility and noise controls, not
authorization. `RestrictPrivateChannels` deliberately turns location into an
additional gate for private-capable commands but does not replace username
authorization.

## Process boundary

- Compiled-in Go tasks/plugins/jobs are trusted and execute in the parent.
- Every file-backed extension invocation is a child process. Lua, JavaScript,
  Gopherbot shell, and Yaegi Go use a versioned stdio RPC protocol; external
  executables use the localhost JSON API.
- The one-shot child model was chosen for crash/hang isolation, short privilege
  lifetime, and simple cleanup. Pooling is acceptable only after measurement
  and must not move policy or brain authority into workers.
- The parent resolves Robot API calls, identity, parameters, secret scope,
  routing, cancellation, and logging. Child interpreters must never receive raw
  config/provider registries, shared encryption state, or privilege tokens.
- Pipeline privilege is fixed by the starter. The composition gate forbidding
  privileged work in an unprivileged pipeline is a security boundary.

`HOME` and `PATH` are host/user environment values. `GOPHER_HOME` names the
robot directory. Direct launcher `GOPHER_*` values are scrubbed from the real
process environment and exposed to work only through engine-controlled
parameters/environment construction.

## Secret scope

An extension may receive a secret only because an administrator attached it
directly, attached its `ParameterSet`, or granted access to the extension's own
brain namespace.

Unprivileged generic methods must not expose configuration objects that permit
secret discovery. Provider registries and credential parameter sets stay
engine-private. Pipeline parameters may flow only according to the privilege
and attachment rules in `getEnvironment`; do not "simplify" this into broad
inheritance.

## UID-only privilege separation

Gopherbot intentionally reverses the usual setuid shape: the executable is
owned/setuid by an unprivileged account, while the parent returns to the
invoking robot UID. A file-backed child then permanently commits before running
extension code to either the robot UID or the unprivileged UID.

There are no ordinary mid-process credential transitions. Privsep changes UID
only; primary GID and supplementary groups are inherited. Therefore:

- grant host privileges by robot UID, not by broad groups;
- do not treat groups as a sandbox boundary;
- keep `.env` owner-readable only;
- keep setgid clear;
- ensure the unprivileged account can traverse the real installed executable
  path, including symlink targets.

Startup fails closed if the self-check child cannot commit to the expected
unprivileged UID. Unsupported platforms run child processes without this UID
boundary and must be treated accordingly.

Setuid behavior requires the manual host validation described in `AGENTS.md`.

## Deliberate limitations

- This is process isolation, not a complete sandbox.
- Trusted compiled Go can access engine memory and host privileges.
- UID-only children retain group access.
- Fine-grained in-language cancellation is secondary to terminating the child
  process.
- External HTTP-backed extensions remain a compatibility surface; new built-in
  runtimes should use the RPC boundary.
