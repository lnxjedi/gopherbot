# Phase 2.5B Impact Surface Report and Onboarding Contract

Date: 2026-09-01

Status: owner-reviewed with revisions; implementation proceeds only through
the separately approved slices recorded in `PLAN.md`.

This proposal converted `PHASE_2_5A_EVIDENCE.md` into the supported new-Robot
contract presented at the Phase 2.5B human gate. The owner-review outcome below
is authoritative where it revises the original proposal.

## Owner review outcome

The owner reviewed the acceptance gate on 2026-09-04 and approved it with
these revisions, which supersede conflicting proposal language below:

- `GOPHER_ENVIRONMENT`, `GOPHER_ENCRYPTION_KEY`,
  `GOPHER_CUSTOM_REPOSITORY`, and `GOPHER_DEPLOY_KEY` are required for a
  configured Robot; `GOPHER_CUSTOM_BRANCH` is the optional fifth launcher
  value and applies to initial clone selection.
- For the Slice 4 startup boundary, blank values count as unset and an absent
  custom repository remains demo mode even when an environment is supplied.
  A nonempty repository requires an explicit valid environment after launcher
  values take precedence over `.env`. Configuration-loading CLI commands have
  the same requirement; `genkey -environment` satisfies it, and no-config CLI
  paths remain exempt.
- The general `env` template helper remains supported. Installed defaults and
  `robot.skel` stop relying on historical `GOPHER_*` values for durable Robot
  behavior, but existing custom configuration may continue reading values such
  as `GOPHER_BOTNAME` or `GOPHER_PROTOCOL` through `env`.
- The scaffold retains onboarding replacement placeholders, concentrated in
  variables files. New-Robot replaces those placeholders with collected user
  values; the mostly static `robot.yaml` and connector files consume the
  resulting named variables rather than receiving copied identity literals.
- Onboarding requires an actual connector-marked direct message from a
  validated administrator. An SSH hidden command issued from a channel does
  not satisfy this narrower onboarding rule; the welcome flow teaches `|c`.
- The version-5 migration proposal is not accepted as written. Because setup
  should be short, implementation should remove general partial-questionnaire
  resumption. The separately approved replacement has one owner and only the
  `encryption-restart`, `repository-handoff`, and `final-restart` durable
  checkpoints; version-4 partial state is rejected rather than migrated.
- Core onboarding has four credential roles: outer encryption key, SSH server
  host key, human SSH public key, and Git deploy key. It does not create a
  general outbound Robot SSH identity. Optional outbound SSH automation may be
  documented separately later.
- `bot-ssh` is retained as the local SSH client helper. New-Robot renames the
  SSH connector's persistent server public key to
  `custom/ssh-host-key.pub`; this is not a general outbound Robot key. The
  helper retains a fallback for existing `custom/robot-ssh.pub` files.
- The deprecated terminal connector configuration is removed from
  `robot.skel`; existing Robots and installed engine defaults are not changed
  by Slice 5.
- The default/demo Robot gives interactive plugins a 35-minute warning and
  42-minute kill window.
  `robot.skel` must explicitly retain `TimeOuts.Plugin.Warn: 7m` and
  `TimeOuts.Plugin.Kill: 14m`, allowing owners to change those values later.
- Main-engine and other independent product changes proceed one at a time,
  with an owner check-in before each slice.

## Proposed outcome

`new robot` becomes a direct-message, single-owner setup conversation with two
restart boundaries:

1. initialize the outer encryption key and restart into encrypted setup; then
2. build and validate the local scaffold, verify that the upstream repository
   can be read with its deploy key, and restart into the configured Robot.

The flow never deletes an existing `custom/` tree, never claims repository
handoff is complete before verification, never stores private credentials in
setup state, and never installs a permanent resume hook into generated Robot
configuration.

Generated `.env` remains the narrow bootstrap file. The public launcher
contract retains four required names plus one optional initial-branch selector:

- `GOPHER_ENVIRONMENT`
- `GOPHER_ENCRYPTION_KEY`
- `GOPHER_CUSTOM_REPOSITORY`
- `GOPHER_DEPLOY_KEY`
- `GOPHER_CUSTOM_BRANCH` only when selecting the first clone branch

Other durable Robot behavior moves to environment config and named
variables/secrets. Engine-created `GOPHER_*` context remains internal to the
execution boundary.

## Impact Surface Report

### Affected subsystems

| Subsystem | Proposed impact |
|---|---|
| Startup and private environment | Define the public launcher allowlist; preserve direct-launcher-over-`.env` precedence and private self-exec handoff; make `.env` updates atomic and compatible with mode `0400` at rest. |
| Startup mode detection | Add an explicit onboarding preflight rather than treating generic `demo` detection as proof that the working directory is disposable. |
| Installed defaults | Remove environment-driven durable behavior from shipped templates; retain only approved bootstrap inputs and engine-owned handoff. |
| Configuration layering | Generate stable identity under `Variables`, encrypted material under `Secrets`, and explicit common/environment-specific files. |
| `robot.skel/` | Replace literal identity substitution and SSH listener env calls with named variables; keep the scaffold delta-only. |
| Onboarding plugin/library | Separate text from logic; enforce private invocation, one active session, safe preflight, explicit state migration, repository verification, and non-destructive recovery. |
| Welcome/resume jobs | Use authoritative configured Robot identity; resume privately; stop copying a temporary resume job into custom config. |
| SSH connector contract | Preserve connector-owned join events, authenticated canonical usernames, direct-message context, server host identity, and message order. |
| Git bootstrap/update | Preserve the narrow deploy key; distinguish initial branch selection from observed runtime branch state; verify remote access before completion. |
| Identity and authorization | Preserve canonical username as the security identity and administrator checks as engine policy. Do not infer identity from display names or transport IDs. |
| Secrets and child execution | Keep encryption and private-key authority in the parent; file-backed onboarding receives only scoped Robot API operations and parameters. |
| Privilege separation | Do not add a privilege transition or expose launcher secrets to children. File work remains under the invoking Robot UID fixed at child start. |
| Compatibility and migration | Introduce an explicit state-schema migration and clear config errors/warnings for retired launcher configuration; update the pre-v3 migration record. |
| User documentation | Update environment/configuration references with the implementation slice; publish the guided onboarding journey only after end-to-end proof and owner field testing. |

### Preserved invariants

- Startup precedence remains deterministic: direct approved launcher input
  overrides `.env`; common variables load before selected-environment
  variables; custom config remains a delta over installed defaults.
- `GOPHER_ENVIRONMENT` selects the same environment for normal startup and CLI
  commands that load Robot configuration.
- Connectors remain authoritative for `DirectMessage`, `HiddenMessage`,
  `SelfMessage`, and validated canonical username.
- Admin and private-command policy run in the engine before onboarding logic.
- The parent retains encryption, authorization, identity, secret scope, and
  privilege authority. A child never receives the outer encryption key or
  deploy private key through broad environment inheritance.
- A primary connector failure remains fatal, secondary connector failures
  remain isolated, and connector-local message order remains preserved.
- Installed extension defaults remain authoritative. Generated custom config
  contains only intentional Robot-specific deltas.
- General outbound SSH automation remains available as an opt-in production
  capability, separate from connector host identity and repository bootstrap.

### Redefined invariants

- `demo` mode alone no longer authorizes filesystem cleanup. New-Robot setup
  requires a dedicated preflight proving that no existing Robot scaffold would
  be overwritten.
- Onboarding has one filesystem owner/session at a time. A JSON map containing
  multiple active sessions is no longer treated as safe merely because writes
  are serialized.
- Setup completion means the local scaffold validates and the configured
  deploy key can read the expected upstream branch. Merely printing git
  commands is not completion.
- `.env` is explicitly writable only during an atomic update and mode `0400`
  at rest. A later onboarding stage may not call `os.WriteFile` directly on
  the read-only file.
- The generated environment selector is always explicit. Production setup
  changes `GOPHER_ENVIRONMENT=development` to
  `GOPHER_ENVIRONMENT=production`; it does not rely on deleting the line and
  falling back implicitly.
- Temporary onboarding state and hooks are absent after completion. A normal
  configured Robot does not carry a copied `resume-setup` job.

### Startup, concurrency, and recovery risks

| Risk | Required control |
|---|---|
| Direct and `.env` values diverge across restart | Preserve the startup environment handoff and test precedence before and after both restarts. |
| Two users mutate one `.env` or scaffold | Enforce one global active session in addition to the existing exclusive update lock. |
| A stale state file suppresses or hijacks setup | Validate version, stage, initiating canonical user, and scaffold fingerprint; remove an empty/completed state file. |
| Existing `custom/` data is destroyed | Remove the automatic recursive cleanup path. Refuse setup with an actionable inventory when a non-owned scaffold exists. |
| Interrupted file writes corrupt bootstrap state | Write a same-directory temporary file with mode `0600`, sync it, atomically rename it, set final mode `0400`, and preserve unrelated approved entries/comments. |
| Setup resumes in a public channel | Require the start command to be private and send resumed prompts with `SendUserMessage`/`PromptUserForReply`, not channel-targeted methods. |
| Join text is mistaken for user identity | Accept resume only from the SSH connector's own self-message join event, then match its authenticated canonical username to the recorded initiator; do not generalize text parsing as an identity mechanism. |
| Repository is empty, unreachable, or grants the wrong key | Wait for a real upstream ref and verify it using the generated deploy credential before final restart. Never print the private deploy key. |
| Local scaffold differs from the pushed repository | Validate the local tree, record the expected branch/commit, and compare it with the remote ref before declaring handoff complete. |
| Restart occurs while a prompt or file mutation is active | Persist the next safe stage only after successful durable writes; request restart only after prompt cleanup and state persistence. |

### Compatibility risks

- Existing Robots use environment calls for identity, provider policy,
  directories, connector settings, and logging. Removal must identify the
  exact retired names and fail with migration guidance rather than silently
  substituting demo defaults.
- `GOPHER_CUSTOM_BRANCH` is currently both a bootstrap input and an
  engine-observed extension parameter. The bootstrap meaning remains public;
  configured-runtime branch observation remains engine-owned. Documentation
  must describe the phase boundary, and tests must prevent launcher input from
  masquerading as observed state after git initialization.
- Version-4 onboarding state contains obsolete stages. The version-5 reader
  must migrate known states once and reject unknown or newer states. It must
  refuse ambiguous multiple-active-session files rather than choosing a user.
- The installed `env` template helper may remain as a general compatibility
  mechanism, but shipped examples and the supported new-Robot path stop using
  it for durable Robot policy. Known retired Gopherbot config inputs receive
  explicit pre-v3 migration diagnostics.
- Existing optional outbound SSH identities are not removed from Robot
  repositories by this work. Only their accidental treatment as a core
  onboarding requirement is removed.

### Documentation impact

Phase 2.5C and 2.5D must update applicable source-adjacent decisions and
migration references in the same logical changes:

- `aidocs/STARTUP_FLOW.md`
- `aidocs/SECRETS_VARIABLES_ENVIRONMENT_DESIGN.md`
- `aidocs/EXECUTION_SECURITY_MODEL.md` if child scope or privileged execution
  changes materially
- `aidocs/TESTING_CURRENT.md`
- `aidocs/V3_COMPATIBILITY_CONTRACT.md`
- `UPGRADING-v3.md` and `CHANGELOG.md`
- installed defaults, scaffold README, deployment examples, and focused tests

The final user-facing guided setup pages remain a Phase 2.5E/Phase 3 product
of proven behavior, but any reference page directly contradicted by an
implementation change must be corrected in the implementation slice.

## Supported first-run contract

### Entry preconditions

`new robot` is accepted only when all of the following are true:

1. the command is invoked in a connector-marked direct message;
2. the connector supplied a validated canonical username;
3. engine authorization recognizes that username as an administrator;
4. no other active onboarding owner exists;
5. no configured custom repository is already active;
6. `custom/conf/robot.yaml` does not exist; and
7. for a brand-new setup session, `custom/` is absent or literally empty. Any
   entry—including hidden files, `.git`, or a symlink—is treated as existing
   data. A resumed active session is handled separately because the first
   restart creates session-owned encryption state under `custom/`.

Failure is non-destructive and names the conflicting condition. Onboarding
does not delete, rename, or merge an unrecognized tree. The owner may move or
remove it outside the Robot and retry. The empty-directory check runs once
before setup state is created and again immediately before `.env` is written,
closing the prompt-time race without claiming ownership of the directory.

### Conversation and restart sequence

1. **Private welcome and encryption choice.** Explain why the outer key is
   needed. Generate one or accept one only through the private prompt. Write
   `.env` atomically with `GOPHER_ENCRYPTION_KEY` and explicit
   `GOPHER_ENVIRONMENT=development`.
2. **Encryption restart.** Persist the initiating canonical user and next
   stage, then restart. The installed demo resume job recognizes the SSH
   connector's join event and continues through a direct user conversation.
3. **Robot and owner profile.** Collect Robot name, full name, email, alias,
   job channel, administrator contact, canonical username, and human SSH public
   key. Confirm the operational meaning of accepted answers without ceremonial
   confirmation steps.
4. **Scaffold.** Copy the minimal skeleton, generate and encrypt the SSH server
   host key through the parent Robot API, add the human public key, write named
   variables and intentional authorization deltas, and validate the generated
   repository before proceeding.
5. **Repository handoff.** Collect the upstream URL, generate the deploy key,
   atomically add the repository and private deploy key to `.env`, and show
   only the public deploy key plus exact git setup commands.
6. **Real dependency wait.** Prompt the owner to continue after the repository
   exists, the public key has read access, and the local scaffold is pushed.
   Timeout or interruption preserves the safe stage without claiming success.
7. **Verification.** Use the deploy credential to read the expected remote
   branch, compare its commit with the local scaffold, and rerun Robot
   validation. A mismatch returns to the same recoverable stage.
8. **Configured restart.** Remove temporary onboarding state, give concise
   clean-directory bootstrap and recovery instructions, and restart into the
   validated local custom config. No onboarding job is written into custom
   config.

The flow may truthfully claim that local configuration and repository handoff
are verified. It may not claim that a separate clean host has been proven;
that remains an explicit Phase 2.5E owner field test.

## Persistence, migration, cancellation, and recovery

### Version-5 checkpoint state

The state file remains `.setup-state`, JSON, atomic, and mode `0600`. It has one
owner and exactly one of three durable checkpoints:

1. `encryption-restart`: `.env` contains the outer key and the first restart is
   pending or complete;
2. `repository-handoff`: scaffold creation completed and repository handoff is
   pending; or
3. `final-restart`: repository handoff reached the existing final-restart
   boundary. Slice 7 will require remote verification before this checkpoint
   can be written.

Only the initiating validated canonical username and, after scaffold creation,
the configured username needed to route the post-restart direct message are
stored. Questionnaire answers, repository values, keys, timestamps, protocol
details, transport IDs, and prompt transcripts are not persisted. The final
instructions recover the nonsecret repository URL from `.env`.

Version-4 partial-questionnaire state is intentionally not migrated. It was
not a public v3 contract, and guessing whether old partial data owns a scaffold
would weaken the non-destructive boundary. An unsupported version stops with
actionable instructions to preserve or manually remove `.setup-state` and any
existing `custom/` data.

### Interruption and cancellation

- Timeout or interruption retains the last durable checkpoint. The short
  questionnaire restarts from its first question rather than persisting
  individual answers.
- `new robot` by the same owner resumes; another owner is refused while the
  session is active.
- Cancellation removes only `.setup-state`. It does not remove `.env`,
  `custom/`, generated public keys already in the scaffold, or an upstream
  repository.
- The cancellation response lists retained artifacts and the explicit manual
  cleanup choices. No destructive cleanup is inferred from `cancel`.
- Completion removes `.setup-state` rather than leaving a nonempty empty-state
  document.

## Configuration and launcher contract

### Generated `.env`

After successful handoff, the file contains exactly the four required
bootstrap values, plus an optional initial branch selector when nondefault:

```text
GOPHER_ENCRYPTION_KEY=...
GOPHER_ENVIRONMENT=development
GOPHER_CUSTOM_REPOSITORY=...
GOPHER_DEPLOY_KEY=...
# GOPHER_CUSTOM_BRANCH=...
```

The example shows shape only; generated or documented output never reveals
actual secret values. Deployments inject the same values through their secret
mechanism or an owner-readable `.env`. Environment selection is explicit in
both development and production.

### Retained public launcher values

| Value | Supported use |
|---|---|
| `GOPHER_ENVIRONMENT` | One-invocation or persisted selection of config, variables/secrets, encrypted data key, and provider behavior. |
| `GOPHER_ENCRYPTION_KEY` | Outer encryption bootstrap. |
| `GOPHER_CUSTOM_REPOSITORY` | Locate an absent custom config checkout. |
| `GOPHER_DEPLOY_KEY` | Authenticate initial and managed repository access. |
| `GOPHER_CUSTOM_BRANCH` | Optional branch selection for the initial clone only. |

### Retired launcher configuration

The supported new-Robot path no longer uses passed-in environment values for
identity, contacts, channels, providers, storage directories, connector
settings, credentials, feature policy, message format, or time zone.

Specifically, shipped configuration stops advertising environment overrides
for Robot name/full name/email/alias, brain/history/provider settings,
state/cache/history/workspace directories, default protocol, job channel,
message format, time zone, SSH host/port, log destination, and log level.
Existing CLI flags remain the one-invocation overrides for log destination,
log level, and SSH port. A future operator need must add a deliberate CLI flag
or pass the owner gate; it does not silently expand this allowlist.

`GOPHER_PROTOCOL`, `GOPHER_HTTP_DEBUG`, `GOPHER_SSH_HOST`, and `GOPHER_PORT`
are not proposed as supported public launcher inputs. Normal behavior belongs
in the selected environment; if a proven recovery/debug journey needs a
one-shot override, the preferred follow-up is a narrowly documented CLI flag
with validation and help.

### Generated variables/config files

`conf/variables/common.yaml` contains named stable Robot values and common
secrets, including:

- `ROBOT_NAME`, `ROBOT_FULL_NAME`, `ROBOT_EMAIL`, and `ROBOT_ALIAS`;
- `DEFAULT_JOB_CHANNEL`; and
- encrypted `SSH_HOST_KEY`.

`conf/variables/development.yaml` and
`conf/variables/production.yaml` exist explicitly, even when initially empty,
so later environment-specific secrets and values have an unambiguous home.
The environment config files continue to select protocol, brain, and logging
policy. Protocol config references named listener variables only when a value
is genuinely reusable; otherwise it uses an intentional literal. The top-level
`-ssh-port` flag remains the launch override.

## Credential and security contract

- **Outer encryption key:** supplied privately, never stored in state or git,
  and available only to startup encryption authority.
- **SSH server host key:** generated for connector server identity; private
  material is encrypted immediately through the parent Robot API and stored as
  a named secret; only its public half is written plainly.
- **Human SSH key:** onboarding accepts only public key material and associates
  it with the validated canonical administrator.
- **Git deploy key:** generated without a passphrase because unattended
  bootstrap must use it; its private encoding is stored only in `.env`, its
  public half is shown for a read-only repository grant, and repository access
  is verified before completion.
- **General outbound Robot identity:** optional production automation, created
  only through explicit configuration/admin workflows and scoped parameters;
  it is absent from the core scaffold.

Onboarding remains a privileged file-backed extension because it writes Robot
configuration, but privilege is fixed at pipeline start and does not imply
root. The child receives no provider registry, broad secret set, outer key, or
deploy key. Encryption and repository credential use occur through explicit
parent-owned operations or narrowly attached privileged tasks.

## English message-catalog contract

### Location and loading

The initial catalog is
`lib/newrobotflow/messages/en.yaml`. It is shipped beside the onboarding
library, loaded from the installed tree, and validated before the welcome or
setup flow sends user-facing text. Version 1 supports English only while using
a locale-ready schema. There is no locale auto-detection or custom-Robot text
override in this slice.

Missing files, duplicate semantic keys, unknown block kinds, unknown pause
classes, missing placeholders, extra placeholders, or malformed YAML disable
the affected onboarding extension with a clear startup/config error. They do
not fall back to terse strings embedded in state-machine code.

### Shape

```yaml
Version: 1
Locale: en
Messages:
  prompt_for_robot_name:
    Blocks:
    - Kind: paragraph
      Text: >-
        Now let's choose the short name people will use to address your Robot.
      PauseAfter: paragraph
    - Kind: prompt
      Text: "What should I call your Robot?"
  repository_commands:
    Blocks:
    - Kind: paragraph
      Text: "Run these commands from `{scaffold_path}`:"
    - Kind: fixed
      Text: |-
        git init
        git add .
```

Stable semantic keys describe conversational purpose, not stage numbers.
Allowed block kinds are `paragraph`, `prompt`, and `fixed`. Placeholders use
explicit named braces and are supplied by typed code at the call site. Pause
classes are named catalog metadata whose durations remain code-owned.

### Boundary

The catalog owns friendly wording, paragraph/list grouping, fixed-width command
blocks, placeholder placement, and pacing selection. Code owns:

- stage order and state changes;
- validation regexes, retry counts, defaults, and canonicalization;
- private/direct targeting and authorization;
- file paths, permissions, writes, deletion policy, and restart requests;
- credential generation, encryption, and repository verification; and
- the choice of which semantic message to send for an outcome.

All onboarding and welcome/resume user-facing prose moves to the catalog. Logs
and developer-facing errors may remain in Go. BasicMarkdown is the normal
message format; `fixed` blocks use the fixed-width API.

## Validation contract

### Focused unit coverage

- launcher-over-`.env` precedence and restart handoff for every retained
  public key;
- `GOPHER_ENVIRONMENT=production gopherbot pull-brain --force` selecting the
  production provider/variables fixture;
- retired shipped env inputs absent from defaults and skeleton;
- common then environment-specific variable precedence;
- atomic `.env` update from mode `0400`, preservation of unrelated approved
  lines/comments, final permissions, and failure cleanup;
- onboarding preflight refusing every nonempty/unowned scaffold case without
  mutation;
- one-owner concurrency and canonical-username matching;
- version-4-to-version-5 migration and rejection of ambiguous/unknown state;
- cancellation artifact behavior;
- catalog schema, semantic-key, placeholder, block, and pause validation;
- private message/prompt method selection; and
- credential-role separation, including absence of private values from state
  and child environments.

### Process-backed coverage

- clean-directory first start through encryption restart;
- private resume on authenticated SSH join;
- scaffold generation followed by `gopherbot validate`;
- interruption and restart at each durable stage;
- repository unavailable, empty, wrong branch, wrong key, mismatched commit,
  and verified-success cases;
- final restart with no copied resume hook and no `.setup-state`;
- bootstrap from a second clean directory using only the documented bootstrap
  inputs; and
- regression coverage for existing Robot startup and update/switch-branch
  pipelines.

### Broader validation and field test

- focused tests, then root and `lib/` module suites;
- `make`, because startup/configuration and installed defaults change;
- applicable process-backed integration selectors through the integration
  runner;
- `helpers/check-docs-hygiene.sh`, `mdbook build docs`, and
  `git diff --check`;
- anonymous validation of representative development, production, and legacy
  Robot configs against the migration diagnostics; and
- owner field test of private first run, both restarts, interruption/recovery,
  repository grant/push/verification, clean-directory bootstrap, and catalog
  voice.

No privsep manual host test is required unless implementation changes privsep
or one of its call sites. The implementation must still prove that file-backed
onboarding receives no broadened secrets or privileges.

## Owner acceptance result

Phase 2.5B owner review accepted these product decisions as revised above:

1. the five-name launcher contract, with `GOPHER_CUSTOM_BRANCH` limited to
   initial clone selection;
2. removal of historical environment-driven Robot behavior from installed
   defaults and the new scaffold while preserving the general `env` template
   helper for custom configuration;
3. direct-message-only, validated-admin onboarding, with the SSH welcome
   teaching the `|c` transition;
4. one active setup owner and non-destructive refusal of existing custom data;
5. two restarts, with upstream branch/commit verification before the second;
6. replacement of general partial-questionnaire resumption with the approved
   three-checkpoint, single-owner state design and no version-4 migration;
7. no permanent custom `resume-setup` hook;
8. four core onboarding credential roles and no generated general outbound
   Robot SSH identity; and
9. the English YAML catalog shape and its strict code/text boundary.

Each independent implementation slice requires an owner check-in before work
begins and another before the next slice starts.
