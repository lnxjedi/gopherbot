# Startup Decisions

The exact call graph is in `bot/start.go`, `bot/bot_process.go`,
`bot/config_load.go`, and `bot/conf.go`. Preserve these ordering constraints:

1. Capture direct launcher `GOPHER_*` values, remove them from the real process
   environment by re-exec, and keep them in the engine-owned environment store.
2. Parse/dispatch help, no-init CLI commands, and internal child commands before
   normal robot initialization.
3. Load `private/environment` or `.env` without overriding launcher values,
   then recompute effective startup mode.
4. Initialize encryption and load pre-connect configuration without executing
   extension code.
5. Validate privsep, initialize the brain, prove cache/lock safety, and replay
   pending cloud writes before modules or connectors.
6. Initialize modules and connector runtimes. Primary connector failure aborts;
   secondary failures remain isolated.
7. Load full configuration, initialize plugins, and wait for the first init
   batch to become quiescent.
8. Start queue providers, open the startup gate, send an optional ready
   message, and signal readiness.

Internal child commands bypass normal startup. Their stdout is protocol data;
diagnostics must go to stderr.

## Configuration precedence

- Launcher values beat private env-file values.
- Installed `conf/` loads before custom `conf/`; custom values override.
- Maps merge recursively, scalars override, lists replace unless an `Append*`
  key is used.
- Custom `conf/variables/common.yaml` loads before
  `conf/variables/<GOPHER_ENVIRONMENT>.yaml`.
- Provider configuration belongs in its provider directory
  (`conf/protocols/`, `conf/brains/`, `conf/history/`, `conf/queues/`), not in
  root `robot.yaml`.
- Pre-connect load must not run external configuration/init code. Post-connect
  load may do so.
- Primary protocol is startup-only. Reload may reconcile secondaries and queue
  providers, then atomically reload connector-local mutable state.

Installed extension defaults remain authoritative. Custom extension files
should specify enablement, local parameters/secrets, and intentional deltas;
copying full defaults creates upgrade drift.

## Shutdown

Stop prompts and queue intake, wait for pipelines, flush brain writes, persist a
released instance lock, stop the brain, then stop connectors and signal
handling. This order prevents new work while preserving durable state and
operator messages.

## Non-obvious modes

- `demo` and `test-dev` may create temporary encryption keys.
- `bootstrap` uses the inert connector until custom configuration is present.
- CLI paths do not start connectors, queues, modules, or the serialized brain
  loop.
- `syntax` and `script` intentionally run without robot configuration.
