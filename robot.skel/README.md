# Gopherbot New Robot Default Configuration

This directory is copied into `custom/` by onboarding workflows (for example `new-robot`).

v3 layout notes:

- `conf/robot.yaml` is the main robot config and includes one environment file:
  `conf/environments/<environment>.yaml` (default `production` via `GOPHER_ENVIRONMENT`).
- Protocol-specific config lives in `conf/protocols/*.yaml`.
- This scaffold is intentionally minimal. Optional connector, plugin, and job
  templates live in the installed `conf/` tree as `*.yaml.sample` files.
- Brain and history provider defaults come from the installed `conf/` tree
  unless your robot explicitly overrides them later.
- The onboarding flow writes local identity + SSH access data into the scaffolded files.
- The scaffold keeps `conf/protocols/terminal.yaml` as an inactive local-development
  convenience, but both shipped environments default to SSH.
- `go.mod` declares `module robot.internal` for local Go plugin/job/task/library development.
- The scaffolded `go.mod` includes commented example `replace` directives for wiring
  `github.com/lnxjedi/gopherbot/robot` and `gopherbot.internal/lib` to a local
  Gopherbot checkout or install tree so editor tooling can resolve imports.

Check upstream periodically for updates:
https://github.com/lnxjedi/gopherbot/tree/main/robot.skel
