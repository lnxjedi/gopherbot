# Deployment Environment Variables

These are the environment variables you are most likely to care about when moving a robot between local development and deployment.

## Usually required in deployed environments

- `GOPHER_ENCRYPTION_KEY`
  - Unlocks encrypted config values and encrypted robot keys.
- `GOPHER_CUSTOM_REPOSITORY`
  - The git URL for the robot repository when bootstrapping from an empty directory.
- `GOPHER_DEPLOY_KEY`
  - A deploy key the robot can use to clone `GOPHER_CUSTOM_REPOSITORY`.

## Commonly useful

- `GOPHER_ENVIRONMENT`
  - Selects `custom/conf/environments/<environment>.yaml` and environment-scoped
    variables and secrets.
  - Defaults to `production` when unset or blank. Startup logs
    `GOPHER_ENVIRONMENT isn't set, defaulting to 'production'` at Info level.
- `GOPHER_CUSTOM_BRANCH`
  - Overrides the branch used for custom config checkout and update flows.

## Variables that matter mostly during local development

- `GOPHER_ENVIRONMENT=development`
  - Explicitly selects development; omitting the value selects production.

For one-run overrides, use `-ssh-port`, `-log`, and `-level`. Durable connector,
logging, storage, and routing choices belong in custom configuration.

## Practical advice

- Put job channel, message format, and time-zone policy in custom
  configuration: `DefaultJobChannel`, `DefaultMessageFormat`, and `TimeZone`.
  Installed defaults are `general`, `BasicMarkdown`, and `America/New_York`;
  they no longer read `GOPHER_JOBCHANNEL`, `GOPHER_MESSAGE_FORMAT`, or
  `GOPHER_TIMEZONE`.
- Keep `.env` mode-restricted and out of casual reach.
- Prefer putting deployment-specific values in `.env` and stable robot behavior in `custom/conf/`.
- Set `PrimaryProtocol` and `DefaultProtocol` in the selected environment's
  configuration; `GOPHER_PROTOCOL` and `GOPHER_DEFAULT_PROTOCOL` no longer
  select installed connector defaults. Runtime `GOPHER_PROTOCOL` still tells
  extensions which connector handled the current message.
