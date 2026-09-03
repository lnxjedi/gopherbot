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
2. Map every page to the approved north-star structure and its DevOps operator
   journeys; resolve content and product-readiness gaps inside the applicable
   later slice rather than maintaining a parallel checklist.
3. Propose an extension to the target table of contents only when the corpus
   exposes a distinct user need that the greenfield design missed; the mere
   existence of a legacy page is not sufficient justification.
4. Produce the final table of contents and clean move/removal disposition
   without making broad content moves or creating transition stubs.
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

### Phase 2.5A: Evidence census and configuration boundary

AI work:

1. Reconcile installed defaults, `robot.skel/`, onboarding extensions and
   shared libraries, startup/configuration paths, focused tests, existing
   onboarding design records, and every locally available Robot repository.
   Synthesize repository evidence into neutral operational categories such as
   production best practices, development patterns, or legacy migration cases;
   never reproduce non-public source identities or secret values.
2. Inventory launcher/private-file environment variables by authority:
   unavoidable pre-configuration bootstrap or host controls; Robot behavior
   that should move to environments plus variables/secrets files; and
   engine-internal child handoff that is not user configuration.
3. Default toward deprecating environment variables as Robot configuration.
   Justify every retained public launcher variable, especially values needed
   before `custom/` can be cloned or decrypted.
4. Audit removed `decrypt` usage, obsolete setup stages and compatibility
   branches, copied defaults, SSH key/passphrase remnants, and other pre-v3
   onboarding cruft. Distinguish connector server host keys, git bootstrap
   deploy credentials, user login public keys, and retired general-purpose
   outbound Robot identities.

### Phase 2.5B: Impact report and onboarding contract

AI work:

1. Produce the required Impact Surface Report for startup, configuration,
   routing, identity, authorization, privacy, secrets, bootstrap credentials,
   privilege separation, migration, compatibility, and documentation.
2. Propose the supported first-run flow, persisted state, restart boundaries,
   recovery behavior, repository handoff, and the exact retained environment
   variable boundary.
3. Propose a locale-ready English message catalog for all user-facing
   onboarding copy. Use stable semantic keys, explicit placeholders, and
   paragraph/list structure; keep validation, matching, state transitions, and
   security logic in code or authoritative configuration.

Human gate:

- Approve the onboarding contract, environment-variable exceptions, migration
  and security tradeoffs, and message-catalog shape before implementation.

### Phase 2.5C: Scaffold and configuration migration

AI work:

1. Update installed defaults, `robot.skel/`, generated variables/secrets files,
   provider files, and migration checks so new Robot configuration is
   delta-only and environment-aware.
2. Put stable Robot identity and other nonsecret Robot-owned values in
   variables files; put environment-specific values in the corresponding
   environment variables file; keep encrypted material in `Secrets`.
3. Remove approved obsolete environment/configuration and SSH identity paths
   with focused validation and explicit migration guidance.

### Phase 2.5D: Flow, state, and message-catalog implementation

AI work:

1. Update the onboarding state machine, resume/recovery behavior, repository
   handoff, and restart transitions to the approved contract.
2. Move user-facing onboarding text into the approved English catalog, add
   missing-key/placeholder validation, and preserve friendly pacing and
   BasicMarkdown behavior.
3. Present the catalog for owner voice editing without requiring logic changes.

### Phase 2.5E: End-to-end validation and handoff

AI work:

1. Run focused unit and process-backed onboarding coverage, configuration and
   documentation checks, clean-directory bootstrap, interrupted-session
   recovery, and repository bootstrap validation.
2. Recheck representative Robot configurations against the migration boundary
   without changing those repositories implicitly; report only neutral
   operational patterns.
3. Update user documentation only after the supported flow is proven.

Human gates:

- Review/edit the English catalog for voice and clarity.
- Field-test the complete first-run, restart, recovery, git handoff, and clean
  bootstrap flow before the onboarding documentation concern is accepted.

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
