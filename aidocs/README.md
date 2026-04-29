Gopherbot is an extensible automation framework designed as a persistent, Go-based chatbot ("robot"). It connects to chat platforms (or terminal) and executes automation pipelines triggered by messages, schedules, and internal calls.

# AI Docs Mission

`aidocs/` exists to make AI onboarding fast and reliable for new implementation work.

Design goals:
- prioritize current operating behavior over historical planning detail
- keep canonical architecture/docs discoverable in a small set of entry points
- keep historical slice artifacts available, but out of the default onboarding path

## Documentation Tiers

### 1) Canonical (default onboarding)

Use these first for current behavior:
- `aidocs/COMPONENT_MAP.md`
- `aidocs/CONNECTOR_CONTRACT.md`
- `aidocs/STARTUP_FLOW.md`
- `aidocs/PIPELINE_LIFECYCLE.md`
- `aidocs/SCHEDULER_FLOW.md`
- `aidocs/EXECUTION_SECURITY_MODEL.md`
- `aidocs/macos-privsep.md`
- `aidocs/setup-style-guide.md`
- `aidocs/GOOGLECHAT_CONNECTOR.md`
- `aidocs/SLACK_CONNECTOR.md`
- `aidocs/SSH_CONNECTOR.md`
- `aidocs/TESTING_CURRENT.md`
- `aidocs/INTEGRATION_HARNESS_PLAN.md`
- `aidocs/V3_COMPATIBILITY_CONTRACT.md`

Project roadmap source of truth:
- root `GOALS_v3.md` (human-maintained)

AI backlog source of truth:
- `aidocs/TODO.md` (AI-maintained “what’s left” tracker)

### 2) Active Workstream Indexes

- `aidocs/multi-protocol/README.md`
- `aidocs/multiprocess/README.md`

These are active entry points only; they should point to canonical docs for current behavior.

### 3) Archive (reference only)

- `aidocs/archive/`

Archive docs are for later reference only if needed (historical rationale, old slice context, migration forensics).
Do not treat archive docs as canonical behavior documentation.

## Navigation Note

References prefer file + symbol anchors (e.g., `bot/handler.go` func `IncomingMessage`) over line-number prose.
Use search/symbol navigation to resolve details quickly.
