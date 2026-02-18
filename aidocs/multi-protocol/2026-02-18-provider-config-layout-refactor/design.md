# Provider Config Layout Refactor (Design)

## Context

Current configuration has connector-local config separated (`conf/protocols/<protocol>.yaml`) but provider-specific config for brain/history still embedded in `conf/robot.yaml` (`BrainConfig`, `HistoryConfig`).

That makes environment-driven robots awkward:
- environment files can cleanly choose `Brain`, but provider-specific settings remain in one large inline conditional block.
- custom robots (including Clu) must keep provider-specific branching logic in `conf/robot.yaml`.

## Design Decision

Adopt provider-scoped configuration files, analogous to protocol files.

### New config directories

- `conf/brains/<brain-provider>.yaml`
  - top-level key: `BrainConfig`
- `conf/history/<history-provider>.yaml`
  - top-level key: `HistoryConfig`

### Ownership model

- `conf/robot.yaml` owns provider selection and global policy:
  - `Brain`
  - `HistoryProvider`
  - identity, security, scheduling, extension config, etc.
- provider files own provider-specific settings only.

### Environment model

Environment includes should set selectors/defaults, not credential blocks.

Example (production):
- `PrimaryProtocol: ssh`
- `DefaultProtocol: slack`
- `Brain: cloudflare`
- `LogDest: stdout`

Example (development):
- `PrimaryProtocol: ssh`
- `DefaultProtocol: ssh`
- `Brain: file`
- `LogDest: stdout`

## Loader Contract and Precedence

### Existing merge semantics retained

Provider file loading uses existing merge behavior from `getConfigFile(...)`:
- installed defaults loaded first
- custom config overlays installed defaults
- maps merge recursively
- slices replace unless key uses `Append*`

### Resolution flow

During `loadConfig(true)`:
1. load and parse `conf/robot.yaml`
2. resolve active `Brain` and `HistoryProvider`
3. load `conf/brains/<Brain>.yaml` and capture `BrainConfig`
4. load `conf/history/<HistoryProvider>.yaml` and capture `HistoryConfig`
5. initialize history provider and later brain provider using these resolved configs

### Key validation

- provider files must fail fast on unknown top-level keys.
- `conf/brains/*.yaml` accepts only `BrainConfig`.
- `conf/history/*.yaml` accepts only `HistoryConfig`.

### Robot schema migration stance

Planned contract (strict, migration-oriented):
- `BrainConfig` and `HistoryConfig` are no longer valid top-level keys in `conf/robot.yaml`.
- robots must move these blocks to provider files.
- this aligns with `aidocs/V3_COMPATIBILITY_CONTRACT.md` (config migration expected, fail-fast preferred).

## Additional Subdirectory Policy

Use additional `conf/` subdirectories when all are true:
1. configuration belongs to a provider selected by a single top-level selector key;
2. settings are provider-specific and not meaningful globally;
3. isolation improves environment-driven config clarity.

Apply now:
- `conf/brains/`
- `conf/history/`

Do not introduce new subdirs in this change for:
- global identity/policy keys (`UserRoster`, `AdminUsers`, etc.)
- connector identity mappings (already in `conf/protocols/*`)
- generic logging/runtime keys

## Non-Goals

- changing extension API signatures
- changing username-authoritative security behavior
- changing message routing semantics
- redesigning provider runtime lifecycles

## Expected Outcomes

- environment files become concise selectors
- provider credentials/config are isolated and easier to reason about
- config structure becomes consistent across protocol/brain/history provider domains
