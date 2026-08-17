# User Documentation Refresh Status

Last updated: 2026-08-17

This is the authoritative resume point for the active project. Verify it
against `git status` before acting, then update it at the next handoff.

## Current position

- Phase: 1 — Mechanical co-location
- State: Human Gate 1 approved; mechanical rendering/integration review passed
- Next owner: AI
- Next action: perform the Phase 1 publishing cutover described under "Next AI
  task" below
- Recommended model/reasoning: GPT-5.6 Terra, medium
- Blocking condition: modifying the separate documentation repository requires
  the applicable filesystem approval; request it when needed rather than
  bypassing the boundary

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

Detailed import evidence is in `IMPORT_VALIDATION.md`.

## Validation state

Passing locally:

- `make docs-check`
- `mdbook build docs` using mdBook 0.5.3
- `git diff --check`
- source/destination file-count and tree comparison
- generated-book byte comparison with the source repository
- workflow YAML, workspace JSON, and changed shell-script syntax checks
- known private robot-instance name scan
- stale sibling-repository integration-reference scan

Not yet run; defer to the cutover/merge workflow unless scope changes:

- GitHub-hosted workflow execution
- full development-container image build
- Pages push
- content-accuracy validation of imported chapters

## Worktree expectation

The main worktree should contain the uncommitted documentation import and
integration changes described above. `docs/book/` should exist after the local
build but remain ignored. The separate documentation worktree should remain
unchanged and clean at the start of the next AI task.

If the actual worktree differs materially, stop and reconcile the difference
before following the next action.

## Next AI task: Phase 1 publishing cutover

Recommended context: GPT-5.6 Terra with medium reasoning. The cutover is
bounded integration work; Sol is unnecessary unless new workflow or
repository-policy conflicts appear.

AI tasks, in order:

1. Re-read root `AGENTS.md`, this status, and the owner's Gate 1 feedback.
2. Verify the main worktree still contains the approved uncommitted import and
   integration changes, and verify the separate documentation worktree is
   clean at the recorded handoff revision.
3. Re-run the Phase 1 validation checks after any intervening changes.
4. Update the old documentation repository so it no longer publishes and its
   README points to the main repository's `docs/` source. This repository is
   outside the normal main-worktree write scope, so request the required
   filesystem approval rather than assuming it.
5. Verify there is one publishing owner and no stale development-container
   references.
6. Update this status with the exact cutover changes, validation results, and
   next owner/action.

The next likely human action will be reviewing the final cutover diff or
merging it, depending on the owner's preferred Git workflow.

## After Phase 1 is merged and cut over

Next owner: AI.

Recommended context: GPT-5.6 Sol with high reasoning for the first information
architecture pass. That pass must classify every page, map gaps to DevOps user
journeys, and propose the complete TOC/move map without broad content rewrites.
The following gate is human approval of audience, support boundaries,
deployment priorities, and navigation.
