# AGENTS.md — Gopherbot

This file provides **persistent instructions** for AI coding agents working in this repository. It defines how to orient yourself, what documents have authority, and how to behave when making changes.

This file is intentionally short and stable. Task-specific instructions should be added temporarily (near the top) and removed once the task is complete.

---

## Required Orientation (Read First)

Before making any code changes, you must read the following documents **in order**:

1. `aidocs/README.md` — routing guide for the codebase
2. `aidocs/COMPONENT_MAP.md` — top-level component and directory map
3. `aidocs/STARTUP_FLOW.md` — authoritative startup and initialization control flow
4. `aidocs/GOALS_v3.md` — intent and evaluation criteria for Gopherbot v3

Do not skip this step. If a proposed change conflicts with these documents, the documents take precedence unless explicitly updated.

---

## Scope and Intent

This repository contains the core Gopherbot engine. Changes should prioritize:

* Internal coherence and explicit control flow
* Reducing reliance on fragile external tools and interpreters
* Improving the extension authoring experience
* Lowering onboarding and re-orientation costs

If you are unsure whether a change advances these goals, stop and explain the tradeoffs before proceeding.

---

## Rules of Engagement

* **Do not guess architecture.** Every claim about behavior must be traceable to specific files, functions, or symbols.
* **Cite anchors.** When explaining or justifying a change, reference concrete file paths and functions.
* **Preserve invariants.** If a change affects startup, configuration loading, or execution order, verify it against `aidocs/STARTUP_FLOW.md`.
* **Be conservative by default.** Prefer minimal, well-scoped changes over large refactors unless explicitly instructed.
* **Ask when uncertain.** Insert `TODO (verify): ...` notes instead of inventing behavior.

---

## Where to Add Documentation

* Agent-oriented orientation, control-flow traces, and system maps belong in `aidocs/`.
* Human-oriented development notes, build instructions, and contributor guidance belong in `devdocs/`.
* Do not mix purposes across these directories.

---

## Task-Specific Instructions

The first task is to get integration tests working, and update `aidocs/` to reflect how the integrated test suite works. Generally: `make test` starts this test suite from the Makefile, and each block of tests starts the "test" connector, sends some commands to a test robot, and expects certain replies and events. At the end of the block, the exit (or quit?) command is sent.

### Post-Task
* Review and update the documentation in `aidocs/` as needed to reflect changes made.

---

## Change Hygiene

* Prefer one logical change per branch.
* Keep documentation changes in the same branch as the code they describe.
* Update agent docs when behavior or structure changes; stale documentation is worse than missing documentation.

---

If these instructions conflict with ad-hoc prompts or assumptions, **this file wins**.
