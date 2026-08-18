# User Documentation Refresh Status

Last updated: 2026-08-17

This is the authoritative resume point for the active project. Verify it
against `git status` before acting, then update it at the next handoff.

## Current position

- Phase: 2A — Greenfield information architecture
- State: North-star proposal revised from owner feedback, including the OAuth
  setup/runtime boundary; final approval pending
- Next owner: Human
- Next action: approve the revised `NORTH_STAR_TOC.md` or identify any final
  correction before the pre-v3 policy update and Phase 2B reconciliation
- Recommended model/reasoning: GPT-5.6 Sol, high
- Blocking condition: owner approval of the north-star structure is required
  before Phase 2B corpus reconciliation

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
classification. After owner approval, Phase 2B will map the corpus and gaps to
that target, propose justified extensions if necessary, and produce the final
move/redirect and disposition map.

The latest Phase 2A correction records that provider-specific OAuth setup may
be complex and plugin-owned, but must converge on generic configuration and
secure long-lived credential storage. Runtime extensions obtain short-lived
credentials through the provider-neutral Robot API and do not own that setup
complexity.

## Next human task: North-star TOC gate

Review the revised navigation and accepted decisions in `NORTH_STAR_TOC.md`.
Approve it as written or identify a final correction. No existing pages will
be classified or moved until this gate is resolved. After approval, AI first
records the relaxed pre-v3 compatibility contract, then begins Phase 2B corpus
reconciliation.

Phase 1 cutover work completed:

1. Confirmed both repositories and the old repository's existing publisher.
2. Removed the old `build.yml` publishing workflow and updated the old README
   to direct contributors to the main repository; the owner committed and
   pushed that cutover.
3. Reconciled the project records with the committed main-repository import.
4. Passed `make docs-check`, `mdbook build docs`, and `git diff --check` in
   both repositories; confirmed a single local publisher and no stale
   development-container reference.
