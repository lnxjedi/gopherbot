# User Documentation Research Baseline

## Imported corpus

At handoff revision `2bd8fb120897eebbcc5f495499fc26718ea5750e`,
the separate repository contains:

- 128 Markdown files under `doc/src/`;
- 105 outside `doc/src/Outdated/`, including `docs/src/SUMMARY.md`;
- 23 explicitly outdated files;
- 70 Markdown destinations in `docs/src/SUMMARY.md`;
- 34 non-outdated content pages not linked from `docs/src/SUMMARY.md`.

The uneven inventory is useful evidence, not an instruction to delete every
orphan. Phase 2 must classify each page deliberately.

## Known source/documentation conflicts

These are starting points for bounded concern reviews, not permission to edit
assertions without source verification:

- The manual's build requirement says Go 1.24 while root `go.mod` declares Go
  1.25.
- Connector credential guidance still shows the removed `decrypt` template
  path instead of environment variable/secret-file guidance.
- The security overview describes an older extension process and privilege
  boundary that conflicts with `aidocs/EXECUTION_SECURITY_MODEL.md`.
- Deployment examples and shipped Kubernetes/systemd assets do not yet use one
  consistent v3 environment-selection story.
- Large configuration and matcher references coexist with very short
  deployment and operations pages; page length is not evidence of operational
  completeness.

## Repository integration surface

Mechanical co-location affects:

- root `README.md` documentation-source language;
- `.github/workflows/` for Pages building and the daily development image;
- `resources/containers/` checkout, image, and workspace paths;
- `helpers/check-docs-hygiene.sh` and the `docs-check` make target;
- contributor/roadmap references that still point to the sibling repository.

The public manual URL and `gh-pages` output branch should remain unchanged.

## Content evidence hierarchy

For each concern, prefer evidence in this order:

1. current source and focused tests;
2. installed `conf/` defaults, `robot.skel/`, and shipped deployment assets;
3. applicable `aidocs/` decision records;
4. current user documentation;
5. historical/outdated documentation only for migration context.

Contradictions between source, defaults, tests, and decision records require a
human product decision or a separately scoped code/config fix. Do not paper
over them with smoother prose.

Slack and Google Chat setup additionally require owner-run platform tests.
Static repository evidence can produce the first draft and verification
checklist, but cannot prove provider-console steps, permissions, credentials,
or current end-to-end onboarding behavior.
