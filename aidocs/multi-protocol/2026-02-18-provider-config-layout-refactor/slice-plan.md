# Slice Plan: Provider Config Layout Refactor

## Implementation Status

- Slice 1: complete
- Slice 2: complete
- Slice 3: complete
- Slice 4: complete
- Slice 5: complete
- Slice 6: complete

## Slice 1: Engine Provider-File Loader

Goal:
- add provider-file loading for brain/history configs with strict key validation.

Scope:
- `bot/conf.go`
- `bot/config_validate.go`
- new helper(s) for provider file loading and validation

Contract:
- resolve `BrainConfig` from `conf/brains/<Brain>.yaml`
- resolve `HistoryConfig` from `conf/history/<HistoryProvider>.yaml`
- unknown keys in provider files fail config load
- inline `BrainConfig`/`HistoryConfig` in `robot.yaml` rejected (strict migration)

Tests:
- new config-load tests for successful provider-file resolution
- fail-fast tests for unknown keys in provider files
- fail-fast tests for legacy inline `BrainConfig`/`HistoryConfig` in `robot.yaml`

## Slice 2: Default `conf/` Migration

Goal:
- migrate installed defaults to provider file layout.

Scope:
- `conf/robot.yaml`
- `conf/brains/file.yaml` (new)
- `conf/brains/mem.yaml` (new)
- `conf/brains/cloudflare.yaml` (new)
- `conf/brains/dynamo.yaml` (new)
- `conf/history/file.yaml` (new)
- `conf/history/mem.yaml` (new)

Expected result:
- installed defaults are complete and valid under new loader contract.

## Slice 3: `robot.skel` Migration (Environment-First)

Goal:
- make scaffolded robots environment-driven via selectors, not inline provider branches.

Scope:
- `robot.skel/conf/robot.yaml`
- `robot.skel/conf/environments/*.yaml`
- `robot.skel/conf/brains/*.yaml` (new)
- `robot.skel/conf/history/*.yaml` (new as needed)
- `robot.skel/README.md`

Expected result:
- environment files primarily choose protocol/default protocol/brain/logging.
- provider credentials/settings live in provider files.

## Slice 4: Clu Migration

Goal:
- align Clu with new provider layout and environment model.

Scope:
- `../clu/custom/conf/robot.yaml`
- `../clu/custom/conf/environments/*.yaml`
- `../clu/custom/conf/brains/*.yaml` (new)
- `../clu/custom/conf/history/*.yaml` (as needed)
- deployment assets in `../clu/resources/` and `../clu/custom/README.md`

Expected result:
- production uses:
  - `PrimaryProtocol: ssh`
  - `DefaultProtocol: slack`
  - `Brain: cloudflare`
- development uses file brain and ssh primary/default.

## Slice 5: Integration Test Fixture Migration

Goal:
- migrate integration test configs to provider file layout.

Scope:
- `test/*/conf/robot.yaml`
- `test/*/conf/brains/*.yaml` (new where provider config required)
- any tests asserting old inline config behavior

Expected result:
- test harness remains deterministic with no reliance on removed inline keys.

## Slice 6: Docs and Upgrade Contract Finalization

Goal:
- publish final migration guidance and invariant updates.

Scope:
- `UPGRADING-v3.md`
- `aidocs/STARTUP_FLOW.md`
- `aidocs/COMPONENT_MAP.md`
- `conf/README.md`

Expected result:
- docs match runtime behavior and migration requirements.

## Suggested Validation Sequence

1. `go test ./bot`
2. `go test ./history/... ./brains/...`
3. `go test ./...`
4. `make test`
5. targeted integration suite (current full-security and JS/Lua/Go full suites as applicable)
