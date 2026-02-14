# New Robot Onboarding Epic (Planning)

Status: planning notes + slice 1/2 implementation tracking

## Why this exists

The old answerfile-based setup path works, but it is high-friction and tied to manual file editing plus legacy assumptions.  
v3 goal is a cleaner, guided path from empty directory to a real robot using normal developer git workflow.

## Confirmed product decisions

- Legacy setup flow does not need to be preserved for new-robot onboarding.
- Add `;new robot` and `;new-robot` entrypoint (same behavior).
- In demo mode, docs should steer user to start and connect as `alice` (or equivalent simple default).
- Ask the user for their canonical username (chat identity), and use that for local ssh login (`bot-ssh -l <username>`).
- For slice 1, persist onboarding workflow state in local `.setup-state` (not brain).
- Newly created robot should start with a persistent file brain by default.
- Use one confirmation gate before applying scaffold changes and triggering restart.
- Do not have robot push git changes on user’s behalf in the new workflow.
- Plugin should guide user through:
  - create/push repo manually
  - add deploy key
  - provide repo clone URL
  - update `.env` accordingly

## Current behavior anchor (what exists now)

- `plugins/welcome.lua` now tells users to run `new robot`.
- `plugins/go-new-robot/new_robot.go` handles onboarding state + slice 2 scaffold apply.
- `plugins/autosetup.sh` implements setup logic.
- `gopherbot init slack` copies `resources/answerfiles/slack.txt` to `answerfile.txt`.
- Running `gopherbot` in setup mode parses answerfile/env values and copies `robot.skel/*` into `custom/`, writes `.env`, generates keys, then restarts.

Relevant files:
- `plugins/welcome.lua`
- `plugins/autosetup.sh`
- `bot/cli_commands.go`
- `resources/answerfiles/slack.txt`
- `robot.skel/`

## Impact Surface Report (Slice 0/Planning)

### 1) Change Summary

- Slice name: New-robot onboarding planning and rollout design
- Goal: define staged migration from answerfile setup to interactive `;new-robot` workflow
- Out of scope:
  - engine/core connector refactors
  - MCP implementation

### 2) Subsystems Affected (expected)

- Onboarding UX/plugins:
  - `plugins/welcome.lua`
  - new onboarding plugin (likely Go external plugin path under `plugins/`)
- Setup/bootstrapping compatibility:
  - `plugins/autosetup.sh` (kept, possibly reduced later)
  - `bot/cli_commands.go` (`init` command messaging)
- Templates/config skeleton:
  - `robot.skel/conf/*.yaml`
  - `.env` generation semantics
- Documentation:
  - `aidocs/GOALS_v3.md` alignment references
  - user docs in `../gopherbot-doc`
  - `devdocs/UPGRADING-v3.md` when behavior shifts

### 3) Current invariants potentially impacted

- Startup sequence must stay deterministic (`detectStartupMode` + pre/post config phases).
- Shared policy/auth decisions remain in engine flows.
- Identity decisions remain username-based; transport IDs remain connector-local.
- Multi-connector isolation must remain intact.
- Config precedence must remain explicit and documented.

### 4) Cross-cutting concerns

- Startup mode transitions (`demo` -> onboarding actions -> restart -> bootstrap/production).
- Config source and precedence (`robot.skel` scaffold vs runtime includes).
- Lifecycle ordering (file writes, key generation, restart timing).
- Prompt timeout behavior for long interactive setup via ssh/terminal.

### 5) Concurrency implications

- Wizard state persistence should be robust across restarts and reconnects.
- Avoid concurrent onboarding sessions clobbering shared files in same robot home.
- Ensure restart-triggered transitions cannot leave half-written config.

### 6) Backward compatibility

- No backward-compatibility requirement for legacy setup in new-robot onboarding.
- Prefer guided `new robot` flow as the supported path.
- Add migration notes to `devdocs/UPGRADING-v3.md` if legacy commands are removed.

### 7) Documentation updates required

- `../gopherbot-doc` onboarding chapters (`botsetup`, `RunRobot`, directory structure references).
- Add "new robot from empty directory" walkthrough using `;new-robot`.

## Proposed phased slices

### Slice 1: Entry UX and wizard shell

- Update welcome text to advertise `;new robot` / `;new-robot`.
- Add new plugin command handlers and minimal state machine:
  - start wizard
  - resume wizard
  - cancel wizard
- Persist wizard state in a local `.setup-state` file in robot home.
- No scaffold write yet, just interaction plumbing and state handling.

Acceptance:
- User can start/resume/cancel from ssh/terminal.
- State survives restart.

### Slice 2: Scaffold creation + local identity

- Implement scaffold operation modeled after existing setup behavior:
  - create/populate `custom/` from `robot.skel`
  - generate required keys/secrets
  - create/update `.env` with generated encryption key and placeholders
- Prompt for canonical username and configure local ssh access accordingly.
- Keep demo mode on in-memory brain for initial onboarding; move to file brain after scaffold is created.

Implementation notes (current):
- `new robot` / `new robot resume` captures canonical username, resolves SSH public key
  (auto-detect `~/.ssh/*.pub` first, prompt fallback), then uses one yes/no confirmation gate.
- Apply step now writes scaffold files and local identity config for SSH `UserMap`,
  plus `.env` defaults (`GOPHER_ENCRYPTION_KEY`, `GOPHER_CUSTOM_REPOSITORY=local`,
  `GOPHER_PROTOCOL=ssh`, `GOPHER_BRAIN=file`).
- Existing legacy `autosetup` remains available for reference only and is marked for later retirement.

Acceptance:
- From empty directory, wizard creates usable scaffold.
- User can restart and connect with `bot-ssh -l <username>`.

### Slice 3: Repository handoff (manual git workflow)

- Guide user through creating remote repo + deploy key upload.
- Collect clone URL and update `.env` (`GOPHER_CUSTOM_REPOSITORY`, deploy key material/vars).
- Explicitly avoid robot-driven git push.

Acceptance:
- User can manually commit/push `custom/` to remote.
- New empty directory + `.env` can bootstrap robot from git.

### Slice 4: Protocol and brain add-on setup commands

- Add follow-up guided commands for:
  - Slack protocol setup
  - Brain provider adjustments
- Keep these as additive workflows after base robot exists.

Acceptance:
- User can progressively configure protocol/brain without manual YAML edits for common cases.

### Slice 5: Legacy setup retirement and cleanup

- Remove legacy setup command references from user-facing docs and welcome text.
- Update migration notes for teams still using answerfile-based setup.

## Open questions to resolve before coding

- Confirm exact prompt list for slice 1+2 (minimum viable questions).
- Refine `.setup-state` locking semantics if parallel onboarding commands become common.
- Decide explicit safety policy for non-empty directories:
  - default: abort with guidance unless user confirms clean/overwrite path.

## Non-goals for this epic start

- No MCP requirement for first implementation.
- No broad engine API expansion unless onboarding plugin proves it is necessary.
