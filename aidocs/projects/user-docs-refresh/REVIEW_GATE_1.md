# Review Gate 1: Mechanical Documentation Import

Status: approved by the owner on 2026-08-17.

## Purpose

Confirm that the co-located book renders and navigates like the previously
published book before changing publishing ownership. This is not a content
accuracy review; known stale material will be handled concern by concern.

## Start the preview

From the Gopherbot repository root:

```bash
mdbook serve docs
```

Open the URL printed by mdBook, normally <http://127.0.0.1:3000>.

## Representative pages

Check these pages:

1. The landing/title page: title, logo, sidebar, and next-page navigation.
2. `Status.html`: ordinary prose and lists.
3. `RobotSetup.html`: commands and onboarding navigation.
4. `config/robot-yaml.html`: a large reference page with nested headings and
   YAML blocks.
5. `reference/SimpleMatcher.html`: long reference content, tables, and code.
6. `api/API-Introduction.html`: internal links to the individual API chapters.
7. `api/Pipeline-API.html`: API formatting and language-specific code blocks.
8. `Security-Overview.html`: navigation to the nested user-approval page.

Also try sidebar navigation and one search query. The generated output is
already byte-identical to the source repository build; this review catches
browser-visible or workflow surprises that file comparison cannot.

## Integration review

In the editor's source-control view, confirm the mechanical shape:

- user-manual source is under `docs/`;
- generated `docs/book/` output is ignored;
- the main repository contains the new docs workflow;
- the development container no longer checks out a second docs repository;
- root `AGENTS.md` contains the documentation-impact and robot-name contracts;
- temporary coordination lives under `aidocs/projects/user-docs-refresh/`.

Do not evaluate whether the chapters accurately describe current v3 behavior
at this gate. That is the purpose of the later evidence/field-test slices.

## Response

Reply with either:

- `Gate 1 approved`, or
- the page/file and issue that should be corrected before cutover.

After approval, the next action is to disable the old publishing workflow,
point the old repository at `docs/`, and begin the table-of-contents evidence
and classification work.
