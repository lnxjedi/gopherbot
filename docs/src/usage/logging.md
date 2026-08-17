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

- `GOPHER_LOGLEVEL`
- `GOPHER_LOGDEST`
- `GOPHER_HTTP_DEBUG`

Be careful with `GOPHER_HTTP_DEBUG`; it is for low-level troubleshooting and may expose sensitive request or response data.
