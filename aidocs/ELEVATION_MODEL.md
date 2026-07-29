# Elevation Decisions

Elevation is proof that the already-authorized canonical user controls the
current session now. It is not permission and connectors do not participate.

- Security order is admin → authorizer → elevator. Scheduled/init/queue work is
  automatic administrator-configured work and skips interactive checks.
- `_elevate` is an engine-reserved callback. The leading-underscore namespace
  is not available to extension-authored matcher commands.
- `ElevatedCommands` may use provider timeout policy;
  `ElevateImmediateCommands` and `Elevate(true)` require fresh confirmation.
- Only `robot.Success` succeeds. Explicit denial, mechanism failure,
  misconfiguration, `robot.Normal`, and all other returns fail closed.
- Successful elevation is remembered for the current pipeline. Provider-level
  reuse windows are separate state owned by the elevator.
- Identity and reuse keys are canonical usernames, never transport IDs.

The plugin model is intentional: TOTP, Duo, and human approval provide
different assurance mechanisms without moving policy into connectors. When
changing provider details, preserve the engine callback contract and add
process-backed success/failure coverage.
