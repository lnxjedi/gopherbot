# Impact Surface Report

## 1) Subsystems Affected (with file paths)

Engine/config loading:
- `bot/conf.go`
- `bot/config_load.go`
- `bot/config_validate.go`
- `bot/handler.go` (provider config consumers unchanged but affected by config source)

Runtime startup (ordering-sensitive):
- `bot/bot_process.go`

Default and template configuration:
- `conf/robot.yaml`
- `conf/brains/*.yaml` (new)
- `conf/history/*.yaml` (new)
- `conf/README.md`
- `robot.skel/conf/robot.yaml`
- `robot.skel/conf/environments/*.yaml`
- `robot.skel/conf/brains/*.yaml` (new)
- `robot.skel/conf/history/*.yaml` (new, optional if provider requires config)

Robot-specific config and deployment docs:
- `../clu/custom/conf/robot.yaml`
- `../clu/custom/conf/environments/*.yaml`
- `../clu/custom/conf/brains/*.yaml` (new)
- `../clu/custom/conf/history/*.yaml` (if used)
- `../clu/resources/*` env/deploy templates referencing protocol env vars
- `../clu/custom/README.md`

Tests:
- `test/*/conf/robot.yaml` (especially filebrain fixtures)
- `test/*/conf/brains/*.yaml` (new where needed)
- `bot/*_test.go` for config loading/validation behavior

Documentation:
- `aidocs/STARTUP_FLOW.md`
- `aidocs/COMPONENT_MAP.md`
- `UPGRADING-v3.md`

## 2) Current Invariants That May Break

Potentially impacted invariants:
- startup sequence deterministic and traceable
- config precedence explicit and documented
- connector isolation when multi-protocol enabled

Mitigation:
- keep all provider config resolution inside existing `loadConfig(true)` pre-connect phase
- reuse `getConfigFile(...)` semantics (no new merge semantics)
- preserve connector/runtime orchestration untouched

## 3) Cross-Cutting Concerns

Startup ordering:
- history provider is initialized during pre-connect load in `loadConfig(true)`
- history config must be resolved before provider initialization
- brain config must remain available before brain initialization in `initBot()`

Config loading model:
- provider files should follow install-default + custom-overlay merge pattern
- selected provider names (`Brain`, `HistoryProvider`) determine provider file path

Validation model:
- `validate_yaml` file-type detection currently treats unknown dirs as robot config
- add explicit handling for `brains/` and `history/` file types

## 4) Concurrency Implications

Low risk.
- changes are in startup/reload config construction paths that already use config locks.
- no new long-lived shared mutable structures required beyond existing `brainConfig` / `historyConfig` raw payloads.

## 5) Backward Compatibility Concerns

Compatibility priorities still honored:
- extension API signatures unchanged
- username-based security unchanged
- persistent brain data compatibility preserved (provider choices and data stores unchanged)

Expected breaking surface:
- config schema migration required:
  - remove inline `BrainConfig`/`HistoryConfig` from `conf/robot.yaml`
  - add provider files under `conf/brains/` and `conf/history/`

Given v3 contract (`aidocs/V3_COMPATIBILITY_CONTRACT.md`), this is acceptable and should be documented in `UPGRADING-v3.md`.

## 6) Documentation Updates Required

Required before/with implementation:
- `aidocs/STARTUP_FLOW.md`
  - add provider-file resolution flow and precedence
- `aidocs/COMPONENT_MAP.md`
  - include new `conf/brains` and `conf/history` conventions
- `UPGRADING-v3.md`
  - migration steps and examples for moved provider config
- `conf/README.md`
  - explain config directory layout and ownership boundaries
- `robot.skel/README.md`
  - template expectations for environment-driven provider selection
