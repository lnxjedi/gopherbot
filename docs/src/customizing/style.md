# Configuration Style Guide

These conventions make robots easier to upgrade and easier for teammates to understand.

## Prefer small custom deltas

Let the engine's shipped defaults stay authoritative. In custom config, override only what your robot truly needs:

- enable or disable features
- add local parameters and secrets
- add local commands
- intentionally redefine behavior

## Keep transport details local to connectors

Put connector-specific identity and credential data in `custom/conf/protocols/<protocol>.yaml`, not in top-level robot config.

## Treat usernames as canonical

The engine's security and policy decisions are username-based. Make connector mappings resolve to the same canonical usernames that exist in `UserRoster`.

## Prefer environments over ad-hoc branching logic

Use `custom/conf/environments/<environment>.yaml` and `GOPHER_ENVIRONMENT` to express development, staging, and production differences cleanly.
