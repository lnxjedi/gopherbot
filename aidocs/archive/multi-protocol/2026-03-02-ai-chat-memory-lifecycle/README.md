# AI Chat Memory Lifecycle (2026-03-02)

This package captures planning artifacts for moving AI chat conversation state from short-term memory APIs to long-term datum storage, with explicit retention and compaction.

Primary goals:
- bounded memory growth
- better token efficiency
- preserved UX continuity for brainstorming/rubber-ducking/codegen workflows
- plugin behavior that is generally useful, not team-specific

Files:
- `design.md` - architecture and contract decisions
- `impact-surface-report.md` - impact/risk analysis
- `slice-plan.md` - implementation slicing plan

Status:
- planning complete
- implementation slices complete
