# User Documentation Import Validation

## Handoff

- Source worktree: clean `gopherbot-doc` `main`
- Source revision: `2bd8fb120897eebbcc5f495499fc26718ea5750e`
- Source files imported: 136 committed files under `doc/`
- Destination files before generated output: 136 under `docs/`
- Generated `doc/book/` output: excluded from the import

## Intentional source differences

The source/destination tree comparison reports only these expected changes:

1. The source repository's development README became `docs/README.md` with
   main-repository build and publishing instructions.
2. `serve.sh` and `cserve.sh` now locate `docs/` from their own path so they can
   be invoked from the repository root.
3. One orphaned historical release instruction was changed to use a neutral
   build-robot description in accordance with the robot-name privacy rule.

All other imported manual source files match the handoff revision. A build with
mdBook 0.5.3 produced byte-identical generated output to the source repository's
current `doc/book/` tree.

## Repository integration changes

- Added a main-repository Pages workflow using the previously pinned mdBook
  version and checksum.
- Removed the separate documentation checkout and `/opt/gopherbot-doc` path
  from the daily development-container workflow and image.
- Changed root and contributor references to point to `docs/`.
- Added `docs/` local-link coverage to the documentation hygiene check.
- Added `/docs/book/` to root `.gitignore`.
- Added root `AGENTS.md` contracts for documentation-impact review,
  robot-instance name privacy, temporary active-project records, and mdBook
  validation.

## Completed checks

- `helpers/check-docs-hygiene.sh` — pass
- `mdbook build docs` with mdBook 0.5.3 — pass
- source/destination manifest count — pass
- generated source/destination tree comparison — identical
- `git diff --check` — pass
- workflow YAML parse — pass
- development workspace JSON parse — pass
- changed shell-script syntax checks — pass
- known private robot-instance name scan across documentation — pass

## Deliberately deferred cutover action

The old repository workflow remains capable of publishing. Disable that
workflow and replace its README with a pointer to the main repository only
after the owner approves the rendered manual and mechanical integration diff.
Until then, do not merge the new main-repository publisher to `main`, because
both repositories would be able to update the same `gh-pages` branch.
