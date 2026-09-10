# Startup Decisions

The exact call graph is in `bot/start.go`, `bot/bot_process.go`,
`bot/config_load.go`, and `bot/conf.go`. Preserve these ordering constraints:

1. Capture direct launcher `GOPHER_*` values, remove them from the real process
   environment by re-exec, and keep them in the engine-owned environment store.
2. Parse/dispatch help, no-init CLI commands, and internal child commands before
   normal robot initialization.
3. Load `private/environment` or `.env` without overriding launcher values,
   then recompute effective startup mode.
4. Validate `GOPHER_ENVIRONMENT` before encryption or configuration loading.
   It has no configured-Robot default; omission is accepted only for demo and
   no-config CLI paths.
5. Initialize encryption and load pre-connect configuration without executing
   extension code.
6. Validate privsep, initialize the brain, prove cache/lock safety, and replay
   pending cloud writes before modules or connectors.
7. Initialize modules and connector runtimes. Primary connector failure aborts;
   secondary failures remain isolated.
8. Load full configuration, initialize plugins, and wait for the first init
   batch to become quiescent.
9. Start queue providers, open the startup gate, send an optional ready
   message, and signal readiness.

Internal child commands bypass normal startup. Their stdout is protocol data;
diagnostics must go to stderr.

## Configuration precedence

- Launcher values beat private env-file values.
- Blank launcher/private values are treated as unset.
- A nonempty `GOPHER_CUSTOM_REPOSITORY` requires an explicit, valid
  `GOPHER_ENVIRONMENT`. With no repository, omission selects demo behavior.
- Configuration-loading CLI commands also require an explicit environment;
  commands that do not load Robot configuration remain exempt, and an explicit
  `genkey -environment` value satisfies that command's requirement.
- Installed `conf/` loads before custom `conf/`; custom values override.
- The installed default `BotInfo` is the fixed Floyd identity. Configured
  Robots override it explicitly from custom configuration rather than through
  launcher environment values.
- The installed default alias is the fixed `;`. Configured Robots override it
  explicitly through custom configuration, normally the `ROBOT_ALIAS` variable.
- Maps merge recursively, scalars override, lists replace unless an `Append*`
  key is used.
- Custom `conf/variables/common.yaml` loads before
  `conf/variables/<GOPHER_ENVIRONMENT>.yaml`.
- Provider configuration belongs in its provider directory
  (`conf/protocols/`, `conf/brains/`, `conf/history/`, `conf/queues/`), not in
  root `robot.yaml`.
- Protocol configuration and load errors are stored together by protocol.
  Missing or invalid configuration must never fall back to another protocol.
  Primary load errors abort startup; secondary load errors remain attached to
  that connector for initialization diagnostics and retry after correction.
- A missing installed or custom file permits the other layer to supply defaults.
  An existing file's read, template, YAML, or serialization error must be
  reported rather than silently using the other layer.
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

- `demo` and `test-dev` may create temporary encryption keys. The installed
  demo configuration also gives interactive plugins a 35-minute warning and
  42-minute kill threshold so first-run setup can remain conversational;
  generated Robot configuration explicitly restores the normal 7-minute and
  14-minute plugin thresholds.
- `bootstrap` uses the inert connector until custom configuration is present.
- CLI paths do not start connectors, queues, modules, or the serialized brain
  loop.
- `syntax` and `script` intentionally run without robot configuration.
