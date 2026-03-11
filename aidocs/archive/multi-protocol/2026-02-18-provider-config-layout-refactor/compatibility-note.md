# Compatibility Note (Slices 1-6 Implemented)

## Change Summary

Implemented:
- move provider-specific brain/history config from `conf/robot.yaml` to provider files:
  - `conf/brains/<provider>.yaml` -> `BrainConfig`
  - `conf/history/<provider>.yaml` -> `HistoryConfig`
- migrate installed defaults, `robot.skel`, and Clu to provider-file layout
- migrate Clu deployment assets and docs to `GOPHER_ENVIRONMENT`-first startup selection

## Compatibility Contract

- extension API signatures remain compatible
- username-based security behavior remains compatible
- persistent brain data compatibility remains prioritized
- configuration schema migration is expected and required

## Breaking Surface

- top-level `BrainConfig` and `HistoryConfig` in `conf/robot.yaml` are invalid
- robots must migrate provider-specific settings into provider files

## Operator Actions

1. keep provider selectors in `conf/robot.yaml` / environment files (`Brain`, `HistoryProvider`)
2. create/update `conf/brains/<Brain>.yaml` with `BrainConfig`
3. create/update `conf/history/<HistoryProvider>.yaml` with `HistoryConfig`
4. validate startup and provider initialization after migration

## Documentation Commitment

Root `UPGRADING-v3.md` must be updated in the same implementation change set.
