# Gopherbot v3 Enhancements

This is the user-visible, high-level view of what Gopherbot v3 improves.
It is intentionally short on internal implementation details.

Status labels:
- `Shipped`: available in current v3 work.
- `In Progress`: actively being implemented.
- `Planned`: agreed direction, not complete yet.

## Shipped

### Multi-Protocol Runtime (`Shipped`)

Run one robot across multiple protocols at the same time.
Primary and secondary protocol behavior is explicit and configurable.

Why it matters:
- one automation brain, many chat surfaces
- protocol failures are isolated where possible
- cleaner routing and operational control

### Username-Authoritative Identity (`Shipped`)

Security decisions remain username-based across protocols (admin users, groups, authorization).
Connector-local IDs are still supported, but connector-specific and implementation-detail level.

Why it matters:
- identity policy is consistent across protocols
- connector-specific transport IDs stay where they belong

### Environment-Driven Robot Configuration (`Shipped`)

Robots can select environment-specific behavior (`development`, `production`, etc.) with includes and provider selectors.
Protocol, brain selection, and runtime defaults are clearer and easier to reason about.

Why it matters:
- easier dev/prod parity
- cleaner onboarding and deployment workflows

### Provider-Scoped Config Layout (`Shipped`)

Brain and history provider settings are organized by provider files (`conf/brains/*`, `conf/history/*`) instead of one large mixed block.

Why it matters:
- reduced config sprawl
- cleaner separation of global policy vs provider-specific credentials/settings

### BasicMarkdown Output Contract (`Shipped`)

Gopherbot now has a standard outgoing message format (`BasicMarkdown`) designed for consistent cross-connector rendering.

Why it matters:
- better cross-protocol readability
- fewer connector-specific formatting surprises

### AI Fallback UX and Rendering Improvements (`Shipped`)

The `go-ai-fallback` plugin has improved streaming behavior and cleaner formatting conversion for chat connectors.
Slack mention rendering reliability has also been improved for common punctuation cases.

Why it matters:
- better real-time interaction quality
- less formatting noise in collaborative conversations

### Durable AI Conversation Lifecycle (`Shipped`)

AI chat context now uses long-term datums with an explicit index, retention pruning, and compaction behavior.

Shipped behavior:
- conversation state stored in datum-backed keys (with legacy short-term read fallback)
- conversation index datum for prune traversal and cleanup
- inactivity retention prune job (`go-ai-prune`) driven by `ScheduledJobs` cron
- deterministic summary + recent-window compaction
- optional model-assisted compaction with deterministic fallback on error

Why it matters:
- avoids unbounded storage growth
- reduces token waste while keeping useful context
- keeps interactive replies resilient when prune/compaction helpers fail

## Planned

### Additional Connector and UX Refinements (`Planned`)

Continue hardening protocol-specific formatting and identity mapping behavior while preserving shared username-based policy semantics.

Why it matters:
- predictable behavior across chat platforms
- less connector-specific friction for users and operators

## Notes

For implementation-level details, see:
- `devdocs/ai-chat.md`
- `aidocs/multi-protocol/2026-03-02-ai-chat-memory-lifecycle/`
