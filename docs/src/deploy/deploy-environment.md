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
  - Selects `custom/conf/environments/<environment>.yaml`.
  - Defaults to `development` for scaffolded robots when not set.
- `GOPHER_CUSTOM_BRANCH`
  - Overrides the branch used for custom config checkout and update flows.
- `GOPHER_SSH_PORT`
  - Overrides the SSH connector listen port.
- `GOPHER_LOGDEST`
  - Overrides the log destination, for example `stdout` or `robot.log`.
- `GOPHER_LOGLEVEL`
  - Sets the runtime log level.
- `GOPHER_MESSAGE_FORMAT`
  - Overrides the default outgoing format. If unset, v3 defaults to `BasicMarkdown`.

## Variables that matter mostly during local development

- `GOPHER_ENVIRONMENT=development`
  - Explicitly pins the development environment if you maintain several.
- `GOPHER_SSH_PORT`
  - Helpful when you run more than one local robot.
- `GOPHER_DEFAULT_PROTOCOL`
  - Useful only for special multi-protocol routing cases.

## Practical advice

- Keep `.env` mode-restricted and out of casual reach.
- Prefer putting deployment-specific values in `.env` and stable robot behavior in `custom/conf/`.
- Do not depend on old top-level `GOPHER_PROTOCOL` habits as your main environment selector; v3 expects `GOPHER_ENVIRONMENT` to drive environment-specific config.
