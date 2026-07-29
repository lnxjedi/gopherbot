# Extension API Decisions

`robot/robot.go` is the canonical Go API. Runtime bridges and libraries are
compatibility adapters, not independent specifications.

## Boundary rules

- Preserve signatures and behavior across v2/v3 unless an API contradicts the
  current security model.
- Extension APIs expose scoped operations, not provider registries, raw
  configuration, shared parameter sets, encryption keys, or other discovery
  paths to secrets.
- Identity credential methods require the caller to have the provider's
  credential `ParameterSet` explicitly attached.
- `SetParameter` primarily supplies subsequent pipeline tasks; immediate
  same-task read-after-write is not the compatibility contract.
- `RaisePriv` is intentionally absent. Privilege is fixed at child/pipeline
  boundaries.

## Adding or changing a Robot method

A method is incomplete until every applicable surface and test is updated:

1. canonical interface and return types in `robot/`;
2. parent implementation and HTTP dispatch in `bot/`;
3. child RPC dispatch;
4. Lua, JavaScript, and GSH bridges;
5. Yaegi exported symbols;
6. Bash, Python, Ruby, and compatibility libraries where supported;
7. focused unit tests and the relevant process-backed runtime suites;
8. migration notes when behavior differs from v2.

Search for an existing analogous method rather than trusting a static file
list; bridge locations evolve. Verify parity by running runtime-tagged
integration selectors through the MCP runner.

Document only intentional gaps or non-obvious semantics here. Do not duplicate
the method catalog from `robot/robot.go`.
