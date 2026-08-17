# AI Documentation

`aidocs/` contains only decisions and operational knowledge that are difficult
to reconstruct safely from source. Read code and tests for types, symbols,
call graphs, configuration fields, and ordinary control flow.

## Core decision records

- `COMPONENT_MAP.md` — ownership boundaries, not a file inventory
- `STARTUP_FLOW.md` — ordering and precedence decisions
- `CONNECTOR_CONTRACT.md` — cross-protocol trust boundary
- `PIPELINE_LIFECYCLE.md` — routing and pipeline semantics
- `EXECUTION_SECURITY_MODEL.md` — authorization, privacy, secrets, and privsep
- `TESTING_CURRENT.md` — required test tools and failure workflow
- `V3_COMPATIBILITY_CONTRACT.md` — what compatibility means

## Scoped decision records

- Connectors: `SLACK_CONNECTOR.md`, `GOOGLECHAT_CONNECTOR.md`,
  `SSH_CONNECTOR.md`
- Extensions: `INTERPRETERS.md`, `EXTENSION_API.md`,
  `EXTENSION_SURFACES.md`, `SIMPLE_MATCHER_DIAGNOSTICS.md`,
  `JS_HTTP_API.md`, `LUA_HTTP_API.md`
- State/integrations: `brain_lock_cache.md`, `OAUTH2_TOKEN_MANAGEMENT.md`,
  `JobQueues.md`, `SCHEDULER_FLOW.md`,
  `SECRETS_VARIABLES_ENVIRONMENT_DESIGN.md`
- UX/operations: `ELEVATION_MODEL.md`, `setup-style-guide.md`,
  `DEV_CONTAINER.md`, `macos-privsep.md`

Root `GOALS_v3.md` is the roadmap. `TODO.md` contains unresolved AI follow-ups
only. Completed plans and slice reports are intentionally absent; use Git
history when historical forensics are actually required.

## Active project records

`projects/README.md` indexes temporary coordination records for work spanning
multiple human/AI review cycles. These records are working state, not permanent
architecture decisions. Remove a project's directory and index entry when its
exit criteria are satisfied; Git history retains the completed record.

Every active project has an authoritative status file named STATUS.md. A fresh
context should read the project README and status first, verify the recorded
state against the worktree, and either perform the named AI action or report
the named human gate. Status records must identify the next owner, exact action,
relevant validation, and current model/reasoning recommendation.
