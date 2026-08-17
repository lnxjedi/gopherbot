# User Documentation Refresh

This temporary project coordinates moving the Gopherbot mdBook into `docs/`
and bringing it current one operational concern at a time. The manual targets
DevOps engineers evaluating, deploying, securing, operating, and extending
Gopherbot.

## Resume here

Read `STATUS.md` first for the current phase, next owner/action, completed
validation, and recommended model/reasoning effort. Then use `PLAN.md` for the
full sequence and the other records for evidence. At every handoff, update
`STATUS.md` so a fresh context can continue without this conversation.

## Source baseline

- Import source: local `gopherbot-doc` repository
- Imported revision: `2bd8fb120897eebbcc5f495499fc26718ea5750e`
- Destination: `docs/`
- Published URL to preserve: `https://lnxjedi.github.io/gopherbot`
- Import rule: committed `doc/` files only; exclude generated `doc/book/`

The import is a source snapshot. The separate repository retains its earlier
history; the import commit records the exact handoff revision.

## Working rules

- One operational concern per branch unless the owner explicitly combines
  concerns.
- Establish source truth before drafting prose.
- Preserve useful current material; do not rewrite merely for uniform style.
- Treat shipped defaults as authoritative and custom configuration as
  delta-only.
- Apply the public/private robot-name rule in root `AGENTS.md` to every draft,
  example, transcript, and source excerpt.
- A polished page is not complete without applicability, prerequisites,
  validation, important failure modes, and security/recovery guidance where
  relevant.

See `PLAN.md` for the staged workflow, `INVENTORY.md` for the research baseline
and known conflicts, `IMPORT_VALIDATION.md` for the mechanical handoff record,
and `REVIEW_GATE_1.md` for the current human review.

## Current stage

Gate 1 is approved. Phase 1's main-repository import and publishing workflow
are committed on `main`; the old repository's publisher has been disabled
locally and is awaiting human review and commit. See `STATUS.md` for the exact
cutover validation and handoff requirements.

## Exit criteria

Remove this project directory and its entry from `aidocs/projects/README.md`
when all of the following are true:

1. `docs/` is the sole maintained source for the published manual.
2. The approved DevOps-oriented table of contents has no unexplained active
   orphans or placeholders.
3. Every planned concern has passed its human field-test/review gate.
4. Documentation build and hygiene checks run in the main repository.
5. Root `AGENTS.md` requires documentation-impact review with source changes.
6. The separate documentation repository no longer publishes competing
   output and clearly points contributors to `docs/`.
