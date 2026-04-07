# Gopherbot Rules

## Build and Test

```bash
make                      # build gopherbot binary
make test                 # unit + integration tests
TEST=JSFull make test     # JS full suite
TEST=LuaFull make test    # Lua full suite
TEST=ShFull make test     # Gopherbot shell full suite
TEST=GoFull make test     # External Go (Yaegi) full suite
RUN_FULL=all make test    # all full suites
```

Always redirect verbose test output to a file, then grep for summary:
```bash
make test > /tmp/gopherbot-test.txt 2>&1; echo "EXIT:$?"
grep -E "^(--- (PASS|FAIL)|FAIL\t|ok\t)" /tmp/gopherbot-test.txt
```

## Orientation

Before proposing or implementing changes, read:
1. `aidocs/README.md` — doc index and tiers
2. `aidocs/COMPONENT_MAP.md` — top-level directory/file map
3. `GOALS_v3.md` — project roadmap

Then load only the aidocs relevant to the task scope.

### Escalated orientation (required when touching these areas)

If a change touches **any** of these, read the full set (`aidocs/README.md`, `COMPONENT_MAP.md`, `STARTUP_FLOW.md`, `GOALS_v3.md`, `TESTING_CURRENT.md`) before coding:

- startup/config load order: `bot/start.go`, `bot/bot_process.go`, `bot/config_load.go`, `bot/conf.go`
- message routing/pipeline ordering: `bot/dispatch.go`, `bot/run_pipelines.go`
- connector runtime/behavior: `connectors/*`, `bot/connector_runtime.go`
- identity/authz semantics: username mapping, roster gates, authorization/elevation
- root/default config structure: `conf/robot.yaml`, `robot.skel/conf/robot.yaml`
- privilege separation or task execution: `bot/privsep.go`, `bot/calltask.go`, `bot/task_execution.go`
- user permission checks: `bot/available.go`, `bot/authorize.go`, `bot/elevate.go`
- pre-pipeline user filtering: `bot/handler.go`

If uncertain, escalate.

For subsystem-specific docs, see `aidocs/README.md` — read the relevant aidocs file before modifying that subsystem.

## Documentation Update Mapping

When behavior changes, update canonical docs in the same change:

| Changed area | Update |
|---|---|
| startup/config loading | `aidocs/STARTUP_FLOW.md` |
| pipeline routing/execution | `aidocs/PIPELINE_LIFECYCLE.md` |
| scheduled job behavior | `aidocs/SCHEDULER_FLOW.md` |
| connector behavior/identity | connector-specific `aidocs/` doc + `COMPONENT_MAP.md` if boundaries moved |
| execution security / privsep | `aidocs/EXECUTION_SECURITY_MODEL.md` |
| extension API/runtime | `aidocs/EXTENSION_API.md` and/or `aidocs/EXTENSION_SURFACES.md` |
| compatibility/config migration | `aidocs/V3_COMPATIBILITY_CONTRACT.md` + `UPGRADING-v3.md` |
| test harness assumptions | `aidocs/TESTING_CURRENT.md` |

Run `helpers/check-docs-hygiene.sh` for any change touching `aidocs/`, `devdocs/`, `AGENTS.md`, or `UPGRADING-v3.md`.

## Change Discipline

- One logical change per branch
- No silent refactors — preserve behavior unless explicitly redefining it
- Planning before implementation for cross-cutting changes
- Extension API behavior compatibility is priority (`aidocs/V3_COMPATIBILITY_CONTRACT.md`)
- Classify every integration test failure as regression vs. intentional change before updating expectations

## Post-Task Checklist

1. Re-validate architectural invariants for the affected subsystem
2. Update required canonical docs (see mapping above)
3. Run applicable tests; redirect output to file, grep summary
4. On failure: classify before fixing expectations
