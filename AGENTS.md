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

For integration test harness and testbot behavior, see `aidocs/TESTING_CURRENT.md`.

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
* **Clarify ambiguity.** If you encounter an unexpected situation or ambiguity (e.g. a missing git branch, an unclear instruction, or multiple plausible behavior-affecting choices), do not make a decision on my behalf. Stop, explain the options and their implications, and ask for clarification.
* **Ask when uncertain.** Insert `TODO (verify): ...` notes instead of inventing behavior.

---

## Where to Add Documentation

* Agent-oriented orientation, control-flow traces, and system maps belong in `aidocs/`.
* Human-oriented development notes, build instructions, and contributor guidance belong in `devdocs/`.
* Do not mix purposes across these directories.

---

## Task-Specific Instructions

-> We're adding a new ssh connector that will be the future default instead of terminal.
* New default ssh connector default to replace terminal connector; this allows a local robot to run in the background and log to stdout, so a future gopherbot-mcp can start a local robot that the developer can connect to.
    * On connect, select initial filter:
        * A(ll) messages - you see every message from every channel and thread
        * C(hannel) messages - you see every message in the channel, including messages in every channel thread
        * T(hread) messages - if you've joined a thread, you see only messages in that thread, if you're in a channel, you only see messages sent to the channel without a thread
    * The built-in ssh connector keeps the 42 most recent messages in a buffer (8k max message size); when a user connects and selects a filter, the connector replays the buffer and sends in order of arrival every message that matches the filter with a timestamp prefix, so the user sees the recent history
### Text rendering
Current (terminal):
```
c:general/u:alice -> ;ping
general: @alice PONG
c:general/u:alice -> ;help ping
general(0005): Command(s) matching keyword: ping
*;ping* - see if the bot is alive
c:general/u:alice -> |j0005
(now typing in thread: 0005)
c:general(0005)/u:alice -> hello world
c:general(0005)/u:alice -> ;ping
general(0005): @alice PONG
```
Future (ssh):
```
(09:15:43)@alice/#general -> ;ping
(09:15:45)@alice/#general(0005) ->;ping
```
### Default configuration
Just as the Terminal Connector has Alice, Bob, Carol, David and Erin, so too will the default ssh connector configuration. The user internal unique ID will be the ssh authorized key hash, like you might find in .ssh/authorized_keys (there can be only one). The default users will have unencrypted keys in resources/ssh-default/alice.key, etc., corresponding to the public key in the default roster. To simplify local dev, `gopherbot/bot-ssh <user>` is a bash wrapper for ssh so the dev can just e.g. "./bot-ssh alice" and connect to the local robot as user alice.

### Listening port
Gopherbot will default to host localhost and port 4221, and exit with error if unable to bind. `--ssh-port NNNN` can override this. When the ssh goroutine binds to the port, it will write a trivial shell env file to $CWD/.ssh-connect:
```
BOT_SSH_PORT=127.0.0.1:4221 (or e.g. 0.0.0.0:4221)
BOT_SERVER_PUBKEY=(...)
```

### Post-Task
* Review and update the documentation in `aidocs/` as needed to reflect changes made.

---

## Change Hygiene

* Prefer one logical change per branch.
* Keep documentation changes in the same branch as the code they describe.
* Update agent docs when behavior or structure changes; stale documentation is worse than missing documentation.

---

If these instructions conflict with ad-hoc prompts or assumptions, **this file wins**.
