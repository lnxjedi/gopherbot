# AGENTS.md — Gopherbot

This is the repository's authoritative AI operating policy. Skills may supply
workflows, but may not override it.

## Tool Policy

- If a required local tool is missing from `PATH`, stop and ask the owner to
  install it. Do not install or substitute tools without approval.
- `.lua`, `.js`, `.gsh`, and interpreted `.go` work does not require a
  standalone runtime. Use `./gopherbot syntax` and `./gopherbot script`; see
  `aidocs/TESTING_CURRENT.md`.

## Orientation

Before proposing or changing code:

1. Read `aidocs/README.md` and `aidocs/COMPONENT_MAP.md`.
2. Read only the decision docs relevant to the task.
3. Inspect the current source and tests; docs intentionally do not duplicate
   source-level control flow or API inventories.

For changes involving startup/configuration, connectors, routing, identity,
authorization, privacy, privilege separation, task execution, root defaults,
or cross-protocol compatibility, also read `aidocs/STARTUP_FLOW.md`,
`aidocs/EXECUTION_SECURITY_MODEL.md`, and `aidocs/TESTING_CURRENT.md`.

Before such a cross-cutting change, share an Impact Surface Report covering:
affected subsystems, preserved/redefined invariants, startup/concurrency and
compatibility risks, validation, and documentation. Do not implement until the
report is shared unless the owner explicitly waives it.

## Architectural Decisions

- Startup and configuration precedence are deterministic and explicit.
- Connectors own transport behavior and accurate message context. The engine
  owns routing, business policy, authorization, elevation, and secret scope.
- Canonical username is the security identity across protocols. Never infer
  identity equivalence from display names or transport IDs.
- A primary connector failure is fatal; secondary connector failures are
  isolated. Preserve message order within each connector.
- Installed extension defaults are authoritative. Custom robot config should
  remain delta-only unless intentionally redefining behavior.
- Unprivileged extensions may receive secrets only through configuration
  explicitly attached to that extension or its authorized brain namespace.
  Never expose provider registries, parameter sets, or broad secret-bearing
  configuration through generic extension methods.

## Security Invariants

### Inbound identity and privacy

- `IgnoreUsers` and `IgnoreUnlistedUsers` are pre-pipeline gates. Keep them
  before worker creation; `IgnoreUsers` matching is case-insensitive.
- Trust a connector-supplied canonical username for policy only when
  `ValidatedUser=true`.
- Connectors are authoritative for `DirectMessage`, `HiddenMessage`, and
  `SelfMessage`; engine/plugins must not rewrite that context.
- Private-command policy is engine-owned and runs before plugin logic.
- `Say`/`Reply` preserve the triggering context; they do not implicitly make a
  response private. Sensitive responses must require private invocation or use
  `Direct()`. Bot-initiated per-user secrets must use `SendUserMessage`.

### Authorization and elevation

- Admin authority has exactly two sources: configured `AdminUsers`, or
  `automaticTask=true`. Automatic tasks are administrator-configured schedules
  and queue triggers; future user-scheduled work must use a separate model.
- Security order is admin → private-context checks → authorizer → elevator.
  Admins bypass the authorizer; elevation is additional assurance after
  authorization.
- `Task.Users` is a whitelist whose empty value permits all users.
- Auth/elevator plugins must explicitly return `robot.Success`; `robot.Normal`
  is a mechanism failure.
- Pipeline elevation persists once achieved. Do not reset it mid-pipeline.

### Execution and privilege separation

- Compiled-in Go extensions are trusted, privileged, in-process engine code.
  File-backed extensions execute in child processes.
- The parent retains all policy, identity, authorization, and secret authority.
  Child interpreters receive only resolved, scoped parameters through the RPC
  boundary.
- There are no normal mid-process privilege transitions. A file-backed child
  commits once, before extension code, to the invoking robot UID or the setuid
  unprivileged UID.
- Privsep is UID-only. GID and supplementary groups are inherited and are not a
  security boundary.
- Pipeline privilege is fixed from its starter. Never add a privileged
  task/job/plugin to an unprivileged pipeline or weaken that gate.

Privsep activates only on a setuid binary and has no normal automated test.
Changes to privsep or its call sites require manual validation: build; install
owned by `nobody` (or platform equivalent) with setuid set and setgid clear; run
as a non-root robot user; verify the `PRIVSEP - UID-only privilege separation
initialized` log includes the expected daemon and unprivileged UIDs; then clear
setuid and restore normal ownership.

## Compatibility and Documentation

Follow `aidocs/V3_COMPATIBILITY_CONTRACT.md`. In short: preserve extension API
and username-security behavior; preserve brain data where feasible; config
schema migration is allowed only when explicit and documented.

For every change to Gopherbot source, installed defaults, `robot.skel/`,
shipped extensions, deployment assets, or user-visible CLI behavior, inspect
`docs/` and decide whether the user documentation is affected. Update affected
user documentation in the same logical change. When no user-doc update is
needed, report that the documentation impact was reviewed. Co-location exists
so source behavior and its user documentation evolve together.

When behavior changes, update the decision document that explains why:

- startup/config precedence: `aidocs/STARTUP_FLOW.md`
- routing/pipelines/schedules/queues: `aidocs/PIPELINE_LIFECYCLE.md`,
  `aidocs/SCHEDULER_FLOW.md`, or `aidocs/JobQueues.md`
- connector identity or transport semantics: `aidocs/CONNECTOR_CONTRACT.md`
  and the connector-specific doc
- execution/security: `aidocs/EXECUTION_SECURITY_MODEL.md`
- extension runtime/API: `aidocs/INTERPRETERS.md` or
  `aidocs/EXTENSION_API.md`
- migration: `aidocs/V3_COMPATIBILITY_CONTRACT.md`, root
  `UPGRADING-v3.md`, and corresponding `conf/` / `robot.skel/` defaults
- test mechanics: `aidocs/TESTING_CURRENT.md`

Only Clu, Floyd, and Bishop are public robot-instance names. Generated or
edited documentation may mention those names. Treat every other
robot-instance name as private: do not reproduce it from source, configuration,
examples, logs, or connected workspaces. Use neutral placeholders such as
`acme-bot` or `example-robot` instead.

`GOALS_v3.md` is the human roadmap. `aidocs/TODO.md` contains only unresolved
AI follow-ups. Active project coordination may live temporarily under
`aidocs/projects/`; each active project must maintain a resumable status file
named STATUS.md with the current phase, completed work, exact next owner/action,
validation state, and recommended model/reasoning effort. Update it at every
human/AI handoff and remove the project records when the project is complete.
Historical slice reports belong in Git history, not `aidocs/`.

Any documentation or AI-instruction change must pass
`helpers/check-docs-hygiene.sh`.
Changes under `docs/` must also pass `mdbook build docs`.

## Change and Validation Discipline

- Keep one logical change per branch unless the owner says otherwise.
- Preserve behavior unless the task explicitly redefines it; document migration
  for intentional changes.
- Revalidate affected invariants and run focused tests before broader tests.
- Rebuild with `make` after core engine or connector-runtime code changes.

When integration coverage applies:

1. Build helpers with `make mcp integration-build` when needed.
2. Use the MCP `run_integration_suite` tool for a specific suite/selector with
   live output disabled. Do not use direct `go test ./test` or
   `make integration-legacy` unless the owner asks for the legacy harness.
3. Start with the compact result; inspect `result.json`, `runner.log`, or
   `robot.log` only as needed.
4. Classify each failure as a regression or an intentional change with stale
   expectations before editing assertions.
5. Report the exact issue for each of 1–3 changed tests; group larger updates.
