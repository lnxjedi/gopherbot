# Gopherbot User Manual

This directory contains the mdBook source for the public manual at
<https://lnxjedi.github.io/gopherbot>.

## Preview locally

From the repository root:

```bash
mdbook serve docs
```

Open the URL printed by mdBook, usually <http://127.0.0.1:3000>.

## Build once

```bash
mdbook build docs
```

Generated output is written to `docs/book/` and is ignored by Git.

## Source and publishing

The manual moved from the separate `gopherbot-doc` repository at revision
`2bd8fb120897eebbcc5f495499fc26718ea5750e`. The main repository now owns the
source and Pages publishing so behavior and user documentation can change
together.

Documentation changes must follow root `AGENTS.md`, including source-truth
review, robot-name privacy, hygiene checks, and mdBook validation.
