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

## Phase 2A: Greenfield information architecture

AI work:

1. Design the ideal DevOps-oriented manual as if no legacy table of contents
   existed.
2. Propose the complete top-level navigation and principal chapters, with the
   user journey and purpose of each major part made explicit.
3. Record the audience, support, deployment-priority, migration, and reference
   boundaries that the proposed structure exposes for owner decision.

Human gate:

- Approve the north-star table of contents, audience, supported deployment
  paths, feature-status language, and navigation before the imported corpus is
  allowed to influence the target structure.

Before Phase 2B, update the authoritative pre-v3 compatibility policy to record
the owner's approved development contract: configuration, extension APIs, and
operational behavior may change before the first v3 release; preserve brain
data where feasible; keep current source, defaults, skeleton, tests, user docs,
and material Changelog entries aligned; establish the stricter public contract
at the v3 release boundary.

## Phase 2B: Corpus reconciliation

AI work:

1. Classify every imported page as keep, rewrite, merge, or remove.
2. Map every page and known content gap to the approved north-star structure
   and its DevOps operator journeys.
3. Propose an extension to the target table of contents only when the corpus
   exposes a distinct user need that the greenfield design missed; the mere
   existence of a legacy page is not sufficient justification.
4. Produce the final table of contents, move/redirect map, gap list, and
   orphan/removal disposition without making broad content moves.
5. Define a pre-cleanup tag before obsolete pages are deleted; do not maintain
   an active documentation archive merely because the imported files existed.

Human gate:

- Approve the corpus reconciliation and final navigation before broad content
  moves or concern-by-concern rewrites.

## Phase 2.5: New-robot scaffold and setup-flow reconciliation

This is a separate implementation context, sequenced after both Phase 2 gates
and before the onboarding documentation concern. It must establish the current
supported onboarding contract before the manual presents a new-robot path as
authoritative.

AI work:

1. Produce the required Impact Surface Report for startup, configuration,
   routing, identity, authorization, privacy, and privilege-separation effects.
2. Reconcile installed defaults, `robot.skel/`, the `new-robot` setup plugin,
   related startup/configuration paths, focused tests, and the existing
   onboarding design records.
3. Identify the supported first-run flow, its persisted state and restart
   boundaries, and any migration or recovery requirements; bring conflicts to
   the owner rather than resolving product policy implicitly.
4. Implement the approved, bounded scaffold/setup-flow changes with focused
   tests, required documentation updates, and owner-run onboarding validation.

Human gate:

- Approve the proposed onboarding contract and any migration/security tradeoffs
  before implementation; field-test the resulting first-run flow before its
  documentation slice is accepted.

## Phase 2.6: Extension-authoring and OAuth readiness

This is a separate implementation context after Phase 2B and before the first
extension tutorial is accepted. A functional robot is expected to contain
custom automation, so extension authoring is part of the primary product path,
not an optional advanced persona.

AI work:

1. Produce an Impact Surface Report for extension runtimes and APIs, parameter
   and secret scope, OAuth identity credentials, privilege, tests, shipped
   examples, and compatibility policy.
2. Specify and build the portable Agent Skills package
   `resources/skills/write-robot-extension/`, covering plugins, jobs, tasks,
   pipelines, libraries, and the shared Robot API across built-in Go, Lua,
   JavaScript, and Gopherbot shell, plus Python as the primary external
   language.
3. Audit authoring source, examples, local checks, and focused integration
   coverage; open bounded product-readiness side stories for behavior that is
   not ready to teach.
4. Field-test the generic OAuth storage/refresh engine and declarative provider
   configuration with GitHub linking, token retrieval, refresh, and a real API
   operation. A maintained provider plugin may own an arbitrarily complex
   setup and consent flow, but it must finish by writing provider-neutral
   configuration and securely storing the required long-lived credential
   material. Ordinary extensions then use the standards-based engine API to
   obtain short-lived credentials for individual API operations. Keep standard
   endpoints, parameters, headers, scopes, and client authentication in
   configuration; add provider code only for setup/authorization UX or
   behavior that cannot be represented by the generic model.
5. Build a tutorial-only JavaScript extension that obtains a short-lived
   per-user GitHub credential through `GetIdentityCredential` and dispatches a
   GitHub workflow.
6. Validate the skill in fresh contexts against representative authoring tasks
   for every targeted language and extension type; improve the skill and
   underlying product/docs when those evaluations expose gaps.

Human gate:

- Exercise the live GitHub OAuth/tutorial path and review representative skill
  outputs before the first-extension and advanced-authoring chapters are
  accepted.

## Phase 3: Concern-by-concern refresh

Recommended concern order:

1. Status, support boundaries, architecture, and requirements.
2. Local SSH demonstration and new-robot onboarding, using the accepted Phase
   2.5 scaffold/setup-flow outcome.
3. Extension model, AI-assisted workflow, and a first credential-free custom
   extension, using the accepted Phase 2.6 skill outcome.
4. Configuration layering, environments, variables, secrets, cloud brains,
   and OAuth identity providers.
5. GitHub OAuth linking and the tutorial-only JavaScript workflow-dispatch
   extension.
6. Dedicated-instance/systemd production deployment, the GitOps update loop,
   cloud-brain requirements, and single-instance replacement patterns.
7. SSH, Slack, Google Chat, and multi-protocol behavior. Slack and Google Chat
   setup are explicitly interactive slices: AI drafts from current source,
   defaults, tests, and existing notes, then the owner must exercise each guide
   against the real platform and provide corrections before acceptance.
8. Cloud-brain persistence, locking, backup, and recovery; file brains are for
   local development and cloud-to-local synchronization, not memorable
   production robots.
9. Identity, privacy, authorization, elevation, and privilege separation.
10. Container and Kubernetes deployment as supported but nonrecommended,
    single-replica models, including shipped artifacts and limitations.
11. Day-two operations: lifecycle, logs, updates, failures, rotation, and
   rollback.
12. Advanced extension authoring across built-in runtimes and Python,
    pipelines, local checks, Robot API usage, and final skill validation.
13. Concise pre-v3 recreation guidance, reference consolidation, a pre-cleanup
    tag, and obsolete-content removal.

Each concern uses this loop:

1. AI produces an evidence brief from current pages, source, tests, defaults,
   skeletons, examples, and relevant decision records.
2. If the intended behavior is not ready to document, AI records a bounded
   product-readiness side story with the required Impact Surface Report,
   validation, documentation impact, and exact resume point. The side story is
   completed before prose presents that behavior as authoritative.
3. Human resolves product-policy questions and identifies real-environment
   validation needs.
4. AI drafts the bounded change and its validation instructions.
5. Human performs the operational/editorial field test.
6. AI incorporates findings, reconciles cross-references, and runs checks.
7. Human accepts and merges the concern.

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
