# User Documentation Refresh Status

Last updated: 2026-08-17

This is the authoritative resume point for the active project. Verify it
against `git status` before acting, then update it at the next handoff.

## Current position

- Phase: 1 — Mechanical co-location
- State: Human Gate 1 approved; local Phase 1 publishing cutover prepared
- Next owner: Human
- Next action: review and commit the old-repository cutover, then confirm the
  main-repository Pages workflow publishes successfully
- Recommended model/reasoning: GPT-5.6 Terra, medium
- Blocking condition: hosted publication cannot be verified until the
  old-repository cutover is committed and the main-repository workflow runs

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
10. Prepared the old repository cutover locally: removed its Pages-publishing
    workflow and redirected its README to the main repository's `docs/` source.

The approved main-repository Phase 1 work is committed and pushed as
`0efdc6b6` (`Start process of moving gopherbot-doc to docs/`). This supersedes
the earlier expectation that the import would remain uncommitted.

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

Not yet run; defer to the cutover/merge workflow unless scope changes:

- GitHub-hosted main-repository Pages workflow execution after cutover review
- full development-container image build
- content-accuracy validation of imported chapters

## Worktree expectation

The main worktree contains the committed Phase 1 import at `0efdc6b6` plus the
uncommitted project-record updates made during this cutover. `docs/book/` may
exist after a local build but remains ignored. The separate documentation
worktree contains the uncommitted publisher-removal and README redirect.

If the actual worktree differs materially, stop and reconcile the difference
before following the next action.

## Next human task: complete the Phase 1 publishing cutover

Review and commit the local old-repository cutover changes, then observe the
main repository's Pages workflow. Confirm the published manual remains at the
preserved URL and that the old repository has no active publishing workflow.
Once confirmed, update this status: Phase 1 is complete and the next owner is
AI for the information-architecture pass.

Local AI work completed for this cutover:

1. Confirmed both repositories and the old repository's existing publisher.
2. Removed the old `build.yml` publishing workflow and updated the old README
   to direct contributors to the main repository.
3. Reconciled the project records with the committed main-repository import.
4. Passed `make docs-check`, `mdbook build docs`, and `git diff --check` in
   both repositories; confirmed a single local publisher and no stale
   development-container reference.

## After Phase 1 is merged and cut over

Next owner: AI.

Recommended context: GPT-5.6 Sol with high reasoning for the first information
architecture pass. That pass must classify every page, map gaps to DevOps user
journeys, and propose the complete TOC/move map without broad content rewrites.
The following gate is human approval of audience, support boundaries,
deployment priorities, and navigation.
