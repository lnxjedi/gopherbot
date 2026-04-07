# New Robot Onboarding Epic (Planning)

Status: planning notes + slice 1/2/3 implementation tracking

## Why this exists

The old answerfile-based setup path works, but it is high-friction and tied to manual file editing plus legacy assumptions.  
v3 goal is a cleaner, guided path from empty directory to a real robot using normal developer git workflow.

## Confirmed product decisions

- Legacy setup flow does not need to be preserved for new-robot onboarding.
- Add `;new robot` and `;new-robot` entrypoint (same behavior).
- In demo mode, docs should steer user to start and connect as `alice` with `bot-ssh -d alice` (or equivalent simple default).
- Ask the user for their canonical username (chat identity), and use that for local ssh login (`bot-ssh <username>`).
- For slice 1, persist onboarding workflow state in local `.setup-state` (not brain).
- Newly created robot should start with a persistent file brain by default.
- After scaffold creation, restart automatically and provide information for pushing the robot to git and testing bootstrap in a new empty directory.
- Keep an encrypted persistent SSH server host key for the robot, but retire legacy `BOT_SSH_PHRASE` / self-managed outbound SSH key setup from the new-robot path.
- `go-new-robot` should not create `custom/binary-encrypted-key` directly; the first restart after writing `GOPHER_ENCRYPTION_KEY` should let the engine create it.
- After that first restart, onboarding should use the robot's `EncryptSecret` method for any remaining encrypted config values needed by the scaffold.
- Do not have robot push git changes on user’s behalf in the new workflow.
- Plugin should guide user through:
  - create/push repo manually
  - add deploy key
  - provide repo clone URL
  - update `.env` accordingly

## Current behavior anchor (what exists now)

- `plugins/welcome.lua` now tells users to run `new robot`.
- `plugins/go-new-robot/new_robot.go` is now the small command entrypoint for starting or canceling onboarding.
- `jobs/go-welcome-join/welcome_join.go` now only handles the initial no-state SSH welcome from
  join announcements (triggered job path), so startup no longer emits welcome chat lines.
- `jobs/go-resume-setup/resume_setup.go` owns reconnect-time onboarding continuation and final
  post-restart bootstrap instructions.
- Legacy answerfile setup automation has been removed from default onboarding flow.

Relevant files:
- `plugins/welcome.lua`
- `bot/cli_commands.go`
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
  - `bot/cli_commands.go` (`init` command messaging)
- Templates/config skeleton:
  - `robot.skel/conf/*.yaml`
  - `.env` generation semantics
- Documentation:
  - root `GOALS_v3.md` alignment references
  - user docs in `../gopherbot-doc`
  - root `UPGRADING-v3.md` when behavior shifts

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
- Add migration notes to root `UPGRADING-v3.md` if legacy commands are removed.

### 7) Documentation updates required

- `../gopherbot-doc` onboarding chapters (`botsetup`, `RunRobot`, directory structure references).
- Add "new robot from empty directory" walkthrough using `;new-robot`.

## Proposed phased slices

### Slice 1: Entry UX and wizard shell

- Update welcome text to advertise `;new robot` / `;new-robot`.
- Add new plugin command handlers and minimal state machine:
  - start wizard
  - cancel wizard
- Persist wizard state in a local `.setup-state` file in robot home.
- No scaffold write yet, just initial interaction plumbing and state handling.

Acceptance:
- User can start/resume/cancel from ssh/terminal.
- State survives restart.

### Slice 2: Scaffold creation + local identity

- Implement scaffold operation modeled after existing setup behavior:
  - create/populate `custom/` from `robot.skel`
  - generate required persistent SSH server host key material
  - create/update `.env` with `GOPHER_ENCRYPTION_KEY` and placeholders
- Prompt for canonical username and configure local ssh access accordingly.
- Keep demo mode on in-memory brain for initial onboarding; move to file brain after scaffold is created.

Implementation notes (current):
- `new robot` only handles the initial encryption-key bootstrap and restart.
- First step writes `.env` with `GOPHER_ENCRYPTION_KEY`, clears any stale `custom/` scaffold state, and restarts.
- That first restart should cause engine startup encryption init to create `custom/binary-encrypted-key`
  automatically from `GOPHER_ENCRYPTION_KEY`.
- After reconnect, the join-triggered resume job applies the scaffold and local identity config for SSH `ProtocolConfig.UserKeys`,
  plus later `.env` defaults such as `GOPHER_CUSTOM_REPOSITORY`.
- Intended cleanup: remove legacy `BOT_SSH_PHRASE` / `robot_key` setup from the onboarding scaffold path.
- Intended cleanup: do not write `custom/binary-encrypted-key` in onboarding code; rely on engine startup to create it on the first restart.
- Intended cleanup: use `EncryptSecret` rather than plugin-local encryption helpers once the first restart has established the real encryption state.
- After scaffold apply, onboarding continues directly into repository handoff in the same resumed flow.
- Legacy answerfile/autosetup flow has been retired.

Acceptance:
- From empty directory, wizard creates usable scaffold.
- User can restart and connect with `bot-ssh <username>`.

### Slice 3: Repository handoff (manual git workflow)

- Guide user through creating remote repo + deploy key upload.
- Collect clone URL and update `.env` (`GOPHER_CUSTOM_REPOSITORY`, deploy key material/vars).
- Explicitly avoid robot-driven git push.

Implementation notes (current):
- No extra phase command is required after reconnect; `resume-setup` resumes the active onboarding session automatically.
- Repo URL is validated, then `.env` is updated with:
  - `GOPHER_CUSTOM_REPOSITORY=<repo-url>`
  - `GOPHER_DEPLOY_KEY=<encoded private deploy key>`
- Deploy keypair is generated during handoff; private key is stored only in parent `.env`
  (encoded for bootstrap), not in `custom/`.
- Public deploy key is written to `custom/ssh/deploy_key.pub` for easy admin copy/paste.
- The persistent SSH server host private key is stored encrypted in config; the corresponding public key is written to `custom/robot-ssh.pub`.
- `custom/ssh/` is only used for the deploy public key during onboarding handoff.
- Deploy key encoding uses legacy bootstrap-compatible format (`space -> _`, `newline -> :`).
- A temporary onboarding resume-on-join hook is enabled during scaffold so the final post-restart bootstrap guidance can run after the robot comes back with its real configuration.
- Wizard replies include the deploy public key plus explicit next-step commands for:
  - manual `git init/add/commit/remote/push`
  - bootstrap verification by restarting from `.env` only.

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
- Document migration notes for teams still using answerfile-based setup.

## Open questions to resolve before coding

- Confirm exact prompt list for slice 1+2 (minimum viable questions).
- Refine `.setup-state` locking semantics if parallel onboarding commands become common.
- Decide explicit safety policy for non-empty directories:
  - default: abort with guidance unless user confirms clean/overwrite path.

## Non-goals for this epic start

- No MCP requirement for first implementation.
- No broad engine API expansion unless onboarding plugin proves it is necessary.
