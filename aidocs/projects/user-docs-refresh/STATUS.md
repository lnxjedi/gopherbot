# User Documentation Refresh Status

Last updated: 2026-09-01

This is the authoritative resume point for the active project. Verify it
against `git status` before acting, then update it at the next handoff.

## Current position

- Phase: 2.5 — New-Robot scaffold and setup-flow reconciliation
- State: Phase 2B approved; Phase 2.5A evidence census complete; Phase 2.5B
  Impact Surface Report and onboarding contract proposed; owner approval
  pending before implementation
- Next owner: human
- Next action: review and approve or revise the nine decisions in the owner
  acceptance gate of `PHASE_2_5B_IMPACT_AND_CONTRACT.md`
- Recommended model/reasoning: GPT-5.6 Sol, high
- Blocking condition: Phase 2.5B human approval is required before implementing
  cross-cutting onboarding changes

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

Not yet run; defer to the cutover/merge workflow unless scope changes:

- full development-container image build
- content-accuracy validation of imported chapters

## Worktree expectation

The main worktree contains the committed Phase 1 import at `0efdc6b6`; it may
also contain current project-plan or handoff-record edits awaiting normal
review. `docs/book/` may exist after a local build but remains ignored. The
separate documentation worktree should be clean at its cutover commit
`908d3a6`.

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
