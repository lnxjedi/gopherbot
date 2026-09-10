# Gopherbot New Robot Default Configuration

This directory is copied into `custom/` by onboarding workflows (for example `new-robot`).

v3 layout notes:

- `conf/robot.yaml` is the main robot config and includes one environment file:
  `conf/environments/<environment>.yaml`, selected by the required
  `GOPHER_ENVIRONMENT` value. There is no configured-Robot default.
- Environment-scoped secrets and deployment values live in
  `conf/variables/common.yaml` and `conf/variables/<environment>.yaml`; these
  files are custom robot data and are not layered from installed defaults.
  New-Robot replaces the placeholders in `common.yaml`; the mostly static
  configuration files read those values with `variable` and `secret`.
- Protocol-specific config lives in `conf/protocols/*.yaml`.
- This scaffold is intentionally minimal. Optional connector, plugin, and job
  templates live in the installed `conf/` tree as `*.yaml.sample` files.
- The environment files explicitly select the file brain, and `robot.yaml`
  explicitly selects file history; change those choices when deploying a
  different provider.
- The onboarding flow writes local identity and SSH access data into the
  scaffolded files. The SSH server public key is `ssh-host-key.pub`; it is not
  an outbound Robot identity.
- Both shipped environments use the SSH connector. The deprecated terminal
  connector is not included in new Robot configuration.
- `go.mod` declares `module robot.internal` for local Go plugin/job/task/library development.
- The scaffolded `go.mod` includes commented example `replace` directives for wiring
  `github.com/lnxjedi/gopherbot/robot` and `gopherbot.internal/lib` to a local
  Gopherbot checkout or install tree so editor tooling can resolve imports.

Check upstream periodically for updates:
https://github.com/lnxjedi/gopherbot/tree/main/robot.skel
