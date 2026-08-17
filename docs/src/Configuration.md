# Configuration Reference

Gopherbot configuration is built from two layers:

- installed defaults from the engine
- custom overrides from your robot

The modern v3 layout is intentionally more structured than older versions:

- `custom/conf/robot.yaml` for robot-wide policy
- `custom/conf/environments/` for environment-specific defaults
- `custom/conf/protocols/` for connector-local config
- `custom/conf/brains/` and `custom/conf/history/` for provider settings
- `custom/conf/plugins/` and `custom/conf/jobs/` for plugin and job metadata

This structure is the backbone of the v3 manual because it is also the backbone of the runtime.

Start with the [`robot.yaml` reference](config/robot-yaml.md) when you need to know where a top-level option belongs or what a robot-wide setting does.
