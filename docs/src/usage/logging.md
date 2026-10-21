# Logging

Gopherbot logs can go to:

- `stdout`
- `stderr`
- a log file such as `robot.log`

## Recommended defaults

- local development: `stdout` is usually fine unless the connector needs a clean terminal
- terminal-style local interaction: let the connector move logs to `robot.log` when needed
- production under `systemd`: plain logs to stdout/stderr are usually easiest to collect

## Useful debug knobs

- `-level debug` or `-level trace` for one run; `LogLevel` in custom config for
  durable policy
- `-log robot.log` for one run; `LogDest` in custom config for durable policy
- `HttpDebug: true` in custom config for low-level HTTP/JSON diagnostics

Installed defaults no longer read `GOPHER_LOGLEVEL`, `GOPHER_LOGDEST`, or
`GOPHER_HTTP_DEBUG`. HTTP debugging is off by default. It does not scrub
sensitive request or response data; when enabling it, select `LogLevel: debug`
and a suitably private log destination explicitly.
