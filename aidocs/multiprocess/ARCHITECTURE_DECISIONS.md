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
