# User Documentation Refresh Plan

Operational state is maintained in `STATUS.md`; this file defines the stable
sequence and gates. A resumed context must not infer the next task from phase
order alone when `STATUS.md` names a human gate, blocker, or partially completed
step.

## Phase 1: Mechanical co-location

AI work:

1. Import the committed mdBook source into `docs/` without content rewrites.
2. Move Pages build/publishing responsibility into the main repository.
3. Remove separate-repository assumptions from the development container,
   workspace, README, roadmap, and contributor references.
4. Extend documentation hygiene to cover local links in `docs/`.
5. Build the book and compare the imported source manifest with the handoff
   revision.

Human gate:

- Review representative rendered pages and the mechanical integration diff.
- Approve the source/publishing cutover before the old publisher is disabled or
  the separate repository is archived.

## Phase 2: Information architecture

AI work:

1. Classify every page as keep, rewrite, merge, archive, or remove.
2. Map pages and gaps to DevOps operator journeys.
3. Propose a complete table of contents and move/redirect map.

Human gate:

- Approve audience, supported deployment paths, feature-status language, and
  navigation before broad content moves.

## Phase 3: Concern-by-concern refresh

Recommended concern order:

1. Status, support boundaries, architecture, and requirements.
2. Local demonstration and new-robot onboarding.
3. Configuration layering, environments, variables, and secrets.
4. systemd production deployment.
5. SSH, Slack, Google Chat, and multi-protocol behavior. Slack and Google Chat
   setup are explicitly interactive slices: AI drafts from current source,
   defaults, tests, and existing notes, then the owner must exercise each guide
   against the real platform and provide corrections before acceptance.
6. Brain persistence, locking, backup, and recovery.
7. Identity, privacy, authorization, elevation, and privilege separation.
8. Container and Kubernetes deployment, including shipped artifacts.
9. Day-two operations: lifecycle, logs, updates, failures, rotation, and
   rollback.
10. v2-to-v3 migration.
11. Extension authoring, pipelines, local checks, and Robot API usage.
12. Reference consolidation and obsolete-content removal.

Each concern uses this loop:

1. AI produces an evidence brief from current pages, source, tests, defaults,
   skeletons, examples, and relevant decision records.
2. Human resolves product-policy questions and identifies real-environment
   validation needs.
3. AI drafts the bounded change and its validation instructions.
4. Human performs the operational/editorial field test.
5. AI incorporates findings, reconciles cross-references, and runs checks.
6. Human accepts and merges the concern.

## Phase 4: Closeout

1. Confirm every active page is intentional and reachable.
2. Confirm the main repository is the only publisher.
3. Point the old repository to the new source and archive it as appropriate.
4. Verify the keep-docs-current contract is operating on ordinary source
   changes.
5. Remove this temporary project directory and its active-project index entry.

## Model and context guidance

Use the least expensive setting that fits the next bounded action, and record
the current recommendation in `STATUS.md`:

- Luna low/medium: manifest checks, mechanical moves, link repair, and other
  fully specified edits.
- Terra medium: ordinary evidence briefs, bounded drafting, corrections, and
  validation.
- Terra high: dense source/config/test reconciliation with limited ambiguity.
- Sol high: information architecture, security or privilege boundaries,
  contradictory evidence requiring judgment, and final audits.

Do not default to `xhigh` or `max`. Re-evaluate these recommendations if the
available model family or project difficulty changes.
