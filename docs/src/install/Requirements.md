# Requirements

## Runtime requirements

- Linux. Gopherbot is developed and tested as a Linux-first tool.
- A shell account where you can run long-lived processes and keep a working directory for each robot.
- `git` and `ssh` if you want bootstrap, update, or repository handoff workflows.
- A team-chat platform or local connector setup appropriate for your robot.

## Build requirements

If you are building from source instead of using a release archive:

- Go 1.24 or newer
- `make`
- standard build utilities such as `tar` and `gzip` if you want packaged archives

## What you do not need for the default v3 workflow

- A dev container just to write plugins
- An extra wrapper script just to start a development robot
- External interpreters for every language up front

You can still use Bash, Python, or Ruby extensions, but the modern path is to start with the built-in interpreters and Go support already included in the engine.
