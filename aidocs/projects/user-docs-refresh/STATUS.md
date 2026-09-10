# User Documentation Refresh Status

Last updated: 2026-09-08

This is the authoritative resume point for the active project. Verify it
against `git status` before acting, then update it at the next handoff.

## Current position

- Phase: 2.5 — New-Robot scaffold and setup-flow reconciliation
- State: Phase 2.5B owner review complete with revisions; implementation
  Slices 1 through 6D plus the separately approved demo-timeout change are
  implemented in the uncommitted working tree; Slice 6D full validation is
  passing and the remainder of the read-only Slice 6 audit is subdivided below
- Next owner: human
- Next action: review completed Slice 6D and approve or revise a separate
  Slice 6E limited to the installed default job channel; all other selectors
  remain separate and unapproved
- Recommended model/reasoning: GPT-5.6 Sol, high
- Blocking condition: owner review is required before the next independent
  product change

The owner requires each independent product change to be implemented and
reviewed separately. `PLAN.md` records the nine current implementation slices.
The Phase 2.5B owner-review addendum records the revised launcher, direct
message, checkpoint, credential, compatibility, and timeout decisions.

Approved Slice 3 design: per-question persistence was replaced with one
single-owner state document and three durable checkpoints only: encryption
written/restart pending, scaffold complete/repository handoff pending, and the
current final-restart boundary. Questionnaire answers are not persisted;
interruption restarts that short questionnaire. The state file is removed
after the configured Robot reconnects. Version-4 partial state is rejected
with actionable manual recovery rather than migrated.

Implemented Slice 4 decision: blank values count as unset; normal startup stays
in demo mode whenever `GOPHER_CUSTOM_REPOSITORY` is absent, whether or not an
environment is supplied. After `.env` loading, a nonempty repository requires
a valid `GOPHER_ENVIRONMENT` before encryption/config loading.
Configuration-loading CLI commands also require an explicit environment,
while `genkey -environment` satisfies that command and
help/version/syntax/script remain exempt. The internal `production` fallback is
removed, and test-development fixtures explicitly select `development`.

The separately approved timeout change gives the default/demo Robot a
35-minute plugin warning and 42-minute kill threshold. `robot.skel` retains
explicit configured-Robot thresholds of `Warn: 7m` and `Kill: 14m`.

Implemented Slice 5 scaffold migration:

- keep onboarding placeholders, but concentrate all varying scalar values in
  `conf/variables/common.yaml`; static scaffold files consume them through
  `variable` and `secret`;
- use `ROBOT_NAME`, `ROBOT_FULL_NAME`, `ROBOT_EMAIL`, `ROBOT_ALIAS`, and
  `DEFAULT_JOB_CHANNEL`, plus the existing encrypted `SSH_HOST_KEY`;
- remove the `GOPHER_ENVIRONMENT` fallback from scaffold `robot.yaml` and make
  SSH listen host/port intentional literals rather than durable environment
  lookups;
- add explicit empty development/production variables files, preserve the
  environment policy files, explain the intended override layering in comments,
  and retain configured-Robot plugin thresholds of 7/14 minutes;
- remove deprecated terminal connector configuration from new scaffolds;
- update onboarding replacement/append logic so placeholders are replaced in
  the variables file rather than copied into static `robot.yaml` or connector
  files;
- retain `bot-ssh` as the local SSH client helper, rename the generated
  persistent server public key to `custom/ssh-host-key.pub`, and preserve an
  existing-Robot fallback for `custom/robot-ssh.pub`; and
- keep installed-default environment-template cleanup separate as Slice 6.

The Slice 6 audit found that the original slice contains multiple independent
behavior changes and must be subdivided. Retain the three approved bootstrap
lookups (`GOPHER_CUSTOM_REPOSITORY`, `GOPHER_DEPLOY_KEY`, and
`GOPHER_CUSTOM_BRANCH`) and preserve the general `env` helper. Separately gate:

1. configured-Robot identity for `welcome-join`/`resume-setup` triggers;
2. ignored legacy top-level content in installed protocol files;
3. root protocol/brain/history/message/time-zone/job-channel selectors;
4. logging and unsafe HTTP-debug selectors;
5. state, brain-cache, brain, and history directory selectors;
6. SSH listener selection and the MCP launcher's current port handoff;
7. provider/sample credential defaults; and
8. deployment assets that still inject retired launcher values.

Implemented Slice 6A is configuration-only and does not add `SelfMessage` to the
job-trigger schema. The installed `welcome-join` and `resume-setup` jobs run
under the fixed Floyd identity before configured startup, while configured
variables are loaded before separately rendering enabled job files. Each
installed onboarding job trigger can therefore choose a template-local default
name of `floyd` in demo mode and replace it with
`variable "ROBOT_NAME"` outside demo mode. `welcome-join` is enabled only in
demo mode; onboarding temporarily enables `resume-setup` in the configured
Robot, where the scaffold variable is then available. This removes the two
`GOPHER_BOTNAME` dependencies without changing the engine, installed-variable
loading rules, or scaffold file inventory.

Implemented Slice 6B removes discarded robot-level identity, authorization,
default-channel, alias, and job-channel keys from the installed null, SSH, and
terminal protocol files. Their `ProtocolConfig` payloads are preserved exactly;
no engine or connector runtime changed. Null-connector cleanup was separately
approved after the initial SSH/terminal boundary was complete.

Implemented Slice 6C replaces only the
installed `conf/robot.yaml` lookups of `GOPHER_BOTNAME` and
`GOPHER_BOTFULLNAME` with the fixed Floyd default identity. Configured Robots
continue overriding `BotInfo` through the scaffold's `ROBOT_NAME` and
`ROBOT_FULL_NAME` variables. `GOPHER_ALIAS` and every protocol, brain, history,
message-format, time-zone, job-channel, logging, directory, listener, provider,
credential, and deployment selector remain outside that slice.

Implemented Slice 6D replaces only the
installed `conf/robot.yaml` lookup of `GOPHER_ALIAS` with the fixed default
alias `;`. Configured Robots continue overriding `Alias` through the
scaffold's `ROBOT_ALIAS` variable. No other selector is included.

Proposed but not approved or implemented Slice 6E would replace only the
installed `conf/robot.yaml` lookup of `GOPHER_JOBCHANNEL` with the fixed default
job channel `general`. Configured Robots would continue overriding
`DefaultJobChannel` through the scaffold's `DEFAULT_JOB_CHANNEL` variable. No
other selector is included.

The owner approved the revised `NORTH_STAR_TOC.md` on 2026-08-31 as the
starting structure for the epic, with the expectation that justified changes
may emerge during later evidence gathering and field testing.

Gate 1 correction during review: the checklist now uses
`api/API-Introduction.html` to test internal links. `api/Pipeline-API.html` has
no internal links and remains in the checklist for its multi-language code
blocks.

Gate 1 was approved by the owner on 2026-08-17 after the correction.

## Completed work

1. Confirmed the separate documentation repository was clean at revision
   `2bd8fb120897eebbcc5f495499fc26718ea5750e`.
2. Imported its 136 committed `doc/` files into `docs/`, excluding generated
   book output.
3. Added the root documentation-impact and robot-instance privacy contracts.
4. Added temporary active-project records and their removal criteria.
5. Added the main-repository mdBook build/Pages publishing workflow.
6. Removed the second documentation checkout and path from the development
   container workflow, image, and workspace.
7. Updated root/contributor references and root `.gitignore`.
8. Extended documentation hygiene checks to active user documentation.
9. Recorded interactive owner field testing as mandatory for Slack and Google
   Chat setup guides.
10. Removed the old repository's Pages-publishing workflow and redirected its
    README to the main repository's `docs/` source.
11. Recorded the approved pre-v3 development boundary in
    `aidocs/V3_COMPATIBILITY_CONTRACT.md` and synchronized the root AI policy,
    roadmap, Changelog, migration guide, and user upgrading page.
12. Classified all 128 Markdown sources: 16 keep, 35 rewrite, 42 merge, and 35
    remove, with no missing or duplicate corpus entries.
13. Produced the concrete final source-tree proposal, complete disposition
    matrix, move/removal policy, and Phase 2B owner questions in
    `PHASE_2B_RECONCILIATION.md`.
14. Received Phase 2B owner approval for the target tree and page dispositions;
    selected clean moves without transition stubs and concern-local handling of
    product-readiness gaps.
15. Completed the Phase 2.5A source/configuration census across startup,
    installed defaults, scaffold/onboarding code and tests, deployment helpers,
    and all seven locally available Robot configuration trees; recorded only
    anonymous operational patterns in `PHASE_2_5A_EVIDENCE.md`.
16. Produced the Phase 2.5B Impact Surface Report and proposed exact
    first-run, launcher, state, recovery, repository, credential, and English
    message-catalog contract in `PHASE_2_5B_IMPACT_AND_CONTRACT.md`.
17. Completed Phase 2.5B owner review, divided implementation into separately
    gated slices, and implemented Slice 1: the engine configuration requires
    administrator/private eligibility, onboarding additionally requires an
    actual direct message, and the SSH welcome teaches `|c` before the
    configured-alias `new robot` command.
18. Implemented the separately approved timeout change: installed demo-mode
    plugin thresholds are `Warn: 35m` and `Kill: 42m`; bootstrap/non-demo
    installed defaults and `robot.skel` remain `Warn: 7m` and `Kill: 14m`.
19. Implemented Slice 2: brand-new onboarding accepts only an absent or
    literally empty `custom/`, checks again before writing `.env`, refuses all
    entries and symlinks non-destructively, and no longer calls `RemoveAll` on
    the scaffold path.
20. Implemented Slice 3: `.setup-state` version 5 now holds one owner and only
    three durable checkpoints; questionnaire answers remain in memory, resume
    messages and prompts are direct, version-4 state is rejected, and cancel or
    successful configured-user reconnect removes the state file without
    changing `.env` or `custom/`.
21. Implemented Slice 4: configured startup validates the explicit
    `GOPHER_ENVIRONMENT` after launcher/private-file precedence is resolved and
    before encryption or config loading; demo/no-config paths remain available,
    CLI config fixtures are explicit, and migration/user documentation records
    the no-default boundary.
22. Implemented Slice 5: New-Robot placeholders are concentrated in common
    named variables, static scaffold configuration consumes those values,
    environment-specific variables files explain their override role, SSH
    listener values are explicit, deprecated terminal configuration is absent,
    and the persistent SSH server public key has an unambiguous filename with a
    compatibility fallback in `bot-ssh`.
23. Implemented Slice 6A: installed onboarding triggers use the fixed Floyd
    identity in demo mode and the configured `ROBOT_NAME` variable after
    `custom/` is active. No job-trigger schema, matcher, connector, or variable
    loader changed.
24. Implemented Slice 6B: installed null, SSH, and terminal files now contain
    only effective connector-owned `ProtocolConfig`; discarded robot-wide
    settings and their otherwise-unused environment lookups were removed.
25. Implemented Slice 6C: installed default `BotInfo` is explicitly Floyd and
    no longer reads `GOPHER_BOTNAME` or `GOPHER_BOTFULLNAME`; configured Robot
    identity remains an explicit custom-variable override.
26. Implemented Slice 6D: the installed default alias is explicitly `;` and no
    longer reads `GOPHER_ALIAS`; configured aliases remain explicit custom
    variable values.

The approved main-repository Phase 1 work is committed and pushed as
`0efdc6b6` (`Start process of moving gopherbot-doc to docs/`). The old
repository cutover is committed and pushed as `908d3a6`
(`Migrated docs to main tree`). This supersedes the earlier expectation that
either cutover worktree would remain uncommitted.

Detailed import evidence is in `IMPORT_VALIDATION.md`.

## Validation state

Passing locally:

- `make docs-check`
- `mdbook build docs` using mdBook 0.5.3
- `git diff --check` in both repositories after the local cutover edits
- source/destination file-count and tree comparison
- generated-book byte comparison with the source repository
- workflow YAML, workspace JSON, and changed shell-script syntax checks
- known private robot-instance name scan
- stale sibling-repository integration-reference scan
- confirmed the old repository has no remaining GitHub Actions workflow files
- confirmed the main repository's `docs.yml` is the sole local publishing
  workflow and development-container references to the old checkout are absent
- GitHub-hosted main-repository Pages workflow execution after cutover, with
  the manual served at the preserved URL
- Phase 2B matrix comparison against the current `docs/src/**/*.md` corpus:
  128 rows for 128 sources, with no missing, extra, or duplicate paths
- focused Phase 2.5A tests:
  `go test ./bot ./jobs/go-resume-setup ./jobs/go-welcome-join`
- onboarding library tests from the separate `lib/` module:
  `GOWORK=off go test ./...`
- Slice 1 focused validation:
  `go test ./bot ./jobs/go-welcome-join ./plugins/go-new-robot`,
  `GOWORK=off go test ./newrobotflow` from `lib/`, and
  `./gopherbot syntax` for both changed interpreted Go extensions
- full root-module/local-script validation for Slice 1: `make test`
- focused timeout configuration tests covering installed demo, installed
  bootstrap/non-demo, and expanded `robot.skel` values: `go test ./bot`
- full root-module/local-script validation after the timeout change:
  `make test` (the sandboxed attempt could not bind the existing Google Chat
  test's loopback listener; the unrestricted rerun passed)
- Slice 2 onboarding-library tests cover absent and empty paths, ordinary and
  hidden entries, `.git`, symlink entries, a symlinked `custom` path, a regular
  file at `custom`, refusal before state/`.env` writes, preservation of data,
  and the second check immediately before `.env`: `GOWORK=off go test
  ./newrobotflow` from `lib/`
- Slice 2 interpreted-plugin and root regression validation:
  `./gopherbot syntax plugins/go-new-robot/new_robot.go`, focused root tests,
  and `make test`
- Slice 3 focused validation: `GOWORK=off go test ./newrobotflow` from `lib/`,
  `./gopherbot syntax` for `go-new-robot` and `go-resume-setup`, and focused
  root tests for both extensions
- Slice 3 full validation: `make test`, `make docs-check`, `mdbook build docs`,
  and `git diff --check`
- Slice 4 focused and full validation: `go test ./bot`, required core rebuild
  with `make`, and `make test` (the restricted attempt could not bind the
  existing Google Chat loopback test; the approved unrestricted rerun passed)
- Slice 4 documentation validation: `make docs-check` and `mdbook build docs`
- Slice 5 generated-scaffold and helper coverage: `go test ./bot .`,
  `GOWORK=off go test ./newrobotflow` from `lib/`, `bash -n bot-ssh`, and
  `GOPHER_ENVIRONMENT=development ./gopherbot validate -redacted-secrets
  robot.skel`
- Slice 5 full root regression validation: `make test`
- Slice 6A focused validation:
  `go test ./bot ./jobs/go-resume-setup ./jobs/go-welcome-join -count=1`
- Slice 6A full and documentation validation: `make test`, `make docs-check`,
  `mdbook build docs`, and `git diff --check`
- Slice 6B focused validation: `go test ./bot -run
  'TestInstalledProtocolDefaultsContainOnlyProtocolConfig' -count=1` and
  `go test ./bot -count=1`
- Slice 6B full and documentation validation: `make test`, `make docs-check`,
  `mdbook build docs`, and `git diff --check`
- Slice 6C focused validation: `go test ./bot -run
  'TestInstalledRobotUsesFixedDefaultIdentity|TestInstalledOnboardingTriggers'
  -count=1`, `rg` confirmation that installed `conf/` has no
  `GOPHER_BOTNAME`/`GOPHER_BOTFULLNAME`, and `go test ./bot -count=1`
- Slice 6C full and documentation validation: `make test`, `make docs-check`,
  `mdbook build docs`, and `git diff --check`
- Slice 6D focused validation: unrestricted `go test ./bot -run
  'TestInstalledRobotUsesFixedDefaultAlias|TestInstalledRobotUsesFixedDefaultIdentity'
  -count=1` after the restricted Go-cache attempt was denied, plus `rg`
  confirmation that installed `conf/` has no `GOPHER_ALIAS`
- Slice 6D full and documentation validation: `make test`, `make docs-check`,
  `mdbook build docs`, and `git diff --check`

Not yet run; defer to the cutover/merge workflow unless scope changes:

- full development-container image build
- content-accuracy validation of imported chapters

## Worktree expectation

The main worktree is expected to contain the uncommitted Slices 1 through 6D,
the separately approved timeout change, their tests, synchronized
user/developer documentation, and project-record changes awaiting owner review.
`docs/book/` may exist after a local build but remains ignored. The separate
documentation worktree should be clean at its cutover commit `908d3a6`.

If the actual worktree differs materially, stop and reconcile the difference
before following the next action.

## Phase 2A output

`NORTH_STAR_TOC.md` proposes the manual that should exist if the imported
corpus did not constrain its structure. It deliberately precedes page-level
classification. The owner approved it on 2026-08-31, after which Phase 2B
mapped the corpus and gaps to that target and produced the proposed final
move/removal and disposition map.

The latest Phase 2A correction records that provider-specific OAuth setup may
be complex and plugin-owned, but must converge on generic configuration and
secure long-lived credential storage. Runtime extensions obtain short-lived
credentials through the provider-neutral Robot API and do not own that setup
complexity.

## Phase 2B result

The approved north-star structure survived corpus reconciliation without a
proposed top-level extension. The corpus exposes substantial content and
product-readiness gaps, but no distinct user journey or operational concern
missing from the Phase 2A spine.

`PHASE_2B_RECONCILIATION.md` expresses the north-star structure as concrete
target paths, classifies every imported Markdown source, and defines the
approved clean-move and removal treatment. Product-readiness gaps are handled
inside their applicable planned slices rather than through a separate list.

The owner approved the reconciliation on 2026-08-31.

## Phase 2.5A result

`PHASE_2_5A_EVIDENCE.md` establishes the source truth for the new-Robot path.
It records `.env` as the supported bootstrap baseline, preserves
`GOPHER_ENVIRONMENT` as a legitimate per-invocation selector, classifies the
remaining launcher and engine-owned environment surfaces, distinguishes all
SSH credential roles, and inventories scaffold/state/recovery gaps.

The anonymous real-Robot comparison found seven available config trees: five
use named identity variables, none yet use per-environment variables files,
and no active removed decrypt template remains. Optional outbound SSH
identities still exist in production automation, but are not core onboarding
requirements.

## Phase 2.5B proposal

`PHASE_2_5B_IMPACT_AND_CONTRACT.md` proposes:

1. a private, validated-admin, single-owner conversation;
2. non-destructive preflight and recovery;
3. two restart boundaries, with remote branch/commit verification before the
   configured restart;
4. a five-name public launcher contract;
5. version-5 single-session state with bounded version-4 migration;
6. no copied custom resume hook;
7. explicit separation of connector, human, deploy, encryption, and optional
   outbound SSH credential roles; and
8. a strict locale-ready English YAML message catalog.

The owner now approves or revises the proposal's nine-decision acceptance gate.
Phase 2.5C implementation must not begin before that handoff.

Phase 1 cutover work completed:

1. Confirmed both repositories and the old repository's existing publisher.
2. Removed the old `build.yml` publishing workflow and updated the old README
   to direct contributors to the main repository; the owner committed and
   pushed that cutover.
3. Reconciled the project records with the committed main-repository import.
4. Passed `make docs-check`, `mdbook build docs`, and `git diff --check` in
   both repositories; confirmed a single local publisher and no stale
   development-container reference.
