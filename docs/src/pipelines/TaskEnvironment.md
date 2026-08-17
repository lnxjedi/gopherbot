# Task Environment

Every task sees a constructed environment, not the raw shell environment of the engine process.

## Sources of task values

From lower priority to higher priority, values can come from:

- engine-provided runtime metadata such as `GOPHER_USER`, `GOPHER_CHANNEL`,
  `GOPHER_PROTOCOL`, and `GOPHER_ENVIRONMENT`
- namespaces
- task, plugin, or job parameters
- pipeline parameters
- `SetParameter(...)` during pipeline execution

## Practical rules

- jobs seed pipeline parameters naturally
- plugins usually have to publish values into the pipeline explicitly with `SetParameter(...)`
- use `GetParameter(...)` for values your extension logic depends on; it reads
  explicit parameters first, then falls back to runtime metadata
- secure parameters must be read with `GetParameter(...)`
- `GOPHER_ENVIRONMENT=development` is the common v3 signal for local
  development and dry-run behavior in plugins that manage host state, external
  systems, or persistent robot memory

The safe mental model is: tasks run with the environment the engine intentionally assembles for them, not whatever happened to be exported in the parent shell.

For fast local checks, `gopherbot script` uses the same `GetParameter(...)`
model with a fixture-backed Robot API; see
[CLI Tools for Authors](../extensiondev/CLI.md).
