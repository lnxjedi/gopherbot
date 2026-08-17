# Connector Credentials

Connector credentials belong in connector-local files under `custom/conf/protocols/`.

That is a deliberate v3 design choice:

- global robot policy stays in `custom/conf/robot.yaml`
- connector identity and transport details stay in each connector's `ProtocolConfig`

## Local development

For local development you often need no third-party credentials at all. The SSH connector is the default local-development connector, and the default robot ships with sample SSH users and keys so you can get started immediately.

## Production connectors

Today the most common production setup is:

- SSH for local development and recovery access
- Slack as the primary team-chat connector

## Credential handling rules

- Keep secrets encrypted in config when possible with `{{ decrypt "..." }}`.
- Keep deployment-specific values in `.env` when they are environment-dependent.
- Keep connector user mappings inside the connector config, not in top-level robot config.

The Slack page on the next step shows the current `Socket Mode` shape.
