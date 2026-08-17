# Troubleshooting

When config behaves strangely, start with the engine's own view of the world.

## Best tools

- `gopherbot validate <path>`
- `gopherbot dump installed robot.yaml`
- `gopherbot dump configured robot.yaml`
- `gopherbot dump configured protocols/ssh.yaml`

## Common causes of trouble

- a value moved from `robot.yaml` into a provider or protocol file in v3
- an old top-level `UserMap` still exists somewhere
- `ProtocolConfig` is missing for the active primary protocol
- a list was overridden when you meant to append, or appended when you meant to replace
- `.env` is missing required deployment values for bootstrap mode

## Operational advice

If reload fails, treat that as a config problem first, not a plugin bug. The merge and validation tools will usually point you to the real issue faster than trial-and-error editing.
