# Multiprocess Architecture Decisions

This document records cross-slice decisions for the multiprocess execution epic.

## 2026-02-14: Child Lifecycle Strategy (Initial)

Decision:
- Start with short-lived child processes (one pipeline/task execution request per child process, then exit).
- Keep all `taskGo` / `bot/*` handlers permanently in-process in the parent.
- Keep brain, authorization, policy, connector state, and routing authority in the parent.
- Do not introduce long-lived privileged child workers in early slices.

Rationale:
- Security first: one-shot children reduce privilege-lifetime and state-leak risk.
- Reliability first: crash/hang isolation is immediate and deterministic.
- Delivery speed: simplest operational model for first production-grade multiprocess slice.
- Future-friendly: execution boundary (`executeTask`) and RPC protocol can later support pooling without redesign.

Performance note:
- Fork/exec overhead is typically small relative to interpreter startup.
- Local spot-check timing on this dev host (rough order-of-magnitude):
  - `/bin/true` ~0.74ms per launch
  - `python3 -V` ~1.6ms per launch
  - `node -v` ~4.3ms per launch
  - `ruby -v` ~10ms per launch
  - `gopherbot --help` ~7.4ms per launch
- Interpretation: for external executable plugins, interpreter startup dominates; one-shot gopherbot children are acceptable for initial slices.

Deferred optimization path (only if needed):
- Add optional unprivileged worker pooling later.
- If pooling is added: enforce strict recycle policies (TTL / max jobs / health checks).
- Keep privileged execution one-shot unless a compelling requirement emerges.

## 2026-02-14: Working Directory Semantics Opportunity

Observation:
- Process-isolated pipeline execution provides a clean path to implement per-pipeline working-directory operations safely.

Implication:
- Child processes can change current working directory (`cwd`) without affecting engine-global process state.
- This avoids the long-standing in-process limitation where thread-local pipeline logic cannot safely mutate process `cwd`.

Follow-up:
- Define explicit pipeline-level working-directory semantics and operator-facing behavior in a dedicated slice.

## 2026-02-14: RPC Scaffold Strategy (Slice 4)

Decision:
- Introduce a minimal versioned stdio protocol scaffold before interpreter migration.
- Keep the initial protocol surface very small (`hello` handshake + `shutdown`) and internal-only.

Rationale:
- De-risks the interpreter migration by validating process startup and protocol framing independently.
- Preserves small-slice safety: no runtime task-path behavior change in scaffold slice.

Follow-up:
- Expand request/response methods incrementally as interpreter-backed task execution moves to child processes.

## 2026-02-14: Generic RPC Protocol, Lua-First Binding (Slice 5)

Decision:
- Keep the RPC transport/protocol generic (request/response envelope plus method dispatch), not Lua-specific.
- Implement Lua as the first interpreter binding on that protocol (`lua_run`, `lua_get_config`, parent-served `robot_call`), and keep `.go`/`.js` migration for later slices.
- Switch Lua execution/config to RPC child mode by default (no fallback flag in this epic branch).

Rationale:
- Preserves one protocol contract for future interpreter migrations.
- Reduces implementation risk by migrating one interpreter first while validating end-to-end behavior under real integration tests.
- Keeps engine policy, authorization, identity mapping, connector routing, and brain operations in the parent process.

Follow-up:
- Add Go and JavaScript interpreter bindings onto the same RPC contract.
- Expand protocol docs/test coverage around timeouts, cancellation, and richer error typing.

## 2026-02-14: JavaScript Binding on Generic RPC (Slice 6)

Decision:
- Add JavaScript as the second interpreter binding on the same generic child RPC contract.
- Keep parent-owned `robot_call` as the single Robot API execution authority for child interpreters.
- Keep external Go (`.go` via yaegi) in-process for now; migrate in a dedicated follow-up slice.

Rationale:
- Reuses the same protocol shape validated by Lua, reducing risk and duplicate transport logic.
- Keeps behavior decomposition clear: transport/protocol is generic, interpreter handlers are language-specific.
- Preserves thin slices and fast fault isolation while maintaining backward-compatible extension behavior.

Follow-up:
- Migrate external Go/yaegi path to generic RPC in its own slice.
- Consider protocol-level cancellation/timeouts and richer error classification once all interpreters are migrated.
