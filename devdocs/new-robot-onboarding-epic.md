# New Robot Onboarding Epic (Planning)

Status: planning-only notes (no engine changes in this session)

## Why this exists

`setup slack`/answerfile works, but it is old, high-friction, and tied to manual file editing plus legacy assumptions.  
v3 goal is a cleaner, guided path from empty directory to a real robot using normal developer git workflow.

## Confirmed product decisions

- Keep old flow available initially; build new flow first.
- Add `;new robot` and `;new-robot` entrypoint (same behavior).
- In demo mode, docs should steer user to start and connect as `alice` (or equivalent simple default).
- Ask the user for their canonical username (chat identity), and use that for local ssh login (`bot-ssh -l <username>`).
- Use a persistent file brain for onboarding workflow state.
- Newly created robot should start with a persistent file brain by default.
- Use one confirmation gate before applying scaffold changes and triggering restart.
- Do not have robot push git changes on user’s behalf in the new workflow.
- Plugin should guide user through:
  - create/push repo manually
  - add deploy key
  - provide repo clone URL
  - update `.env` accordingly

## Current behavior anchor (what exists now)

- `plugins/welcome.lua` tells users to run `setup slack`.
- `plugins/autosetup.sh` implements setup logic.
- `setup slack`/`gopherbot init slack` copies `resources/answerfiles/slack.txt` to `answerfile.txt`.
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
  - removing legacy setup path in v3.0

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

- Keep `setup slack` and answerfile flow working in v3.0.
- Introduce new flow as preferred path; mark legacy path as compatibility.
- Add migration notes to `devdocs/UPGRADING-v3.md` before deprecating/removing legacy behavior.

### 7) Documentation updates required

- `../gopherbot-doc` onboarding chapters (`botsetup`, `RunRobot`, directory structure references).
- Add "new robot from empty directory" walkthrough using `;new-robot`.
- Preserve legacy appendix while migration window is open.

## Proposed phased slices

### Slice 1: Entry UX and wizard shell

- Update welcome text to advertise `;new robot` / `;new-robot`.
- Add new plugin command handlers and minimal state machine:
  - start wizard
  - resume wizard
  - cancel wizard
- Persist wizard state in file brain-backed memory keys.
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
- Keep brain default as persistent file brain.

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

### Slice 5: Legacy path deprecation plan (v3.1+)

- Add warnings and docs migration timeline.
- Keep removal deferred until adoption signal is clear.

## Open questions to resolve before coding

- Confirm exact prompt list for slice 1+2 (minimum viable questions).
- Decide where wizard lock/state metadata lives (brain-only vs lock file + brain).
- Decide explicit safety policy for non-empty directories:
  - default: abort with guidance unless user confirms clean/overwrite path.

## Non-goals for this epic start

- No MCP requirement for first implementation.
- No broad engine API expansion unless onboarding plugin proves it is necessary.
