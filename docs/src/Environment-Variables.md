# Environment Variables

Environment variables influence both startup and task execution.

## Startup values you will use most often

- `GOPHER_ENCRYPTION_KEY`
- `GOPHER_CUSTOM_REPOSITORY`
- `GOPHER_DEPLOY_KEY`
- `GOPHER_CUSTOM_BRANCH`
- `GOPHER_ENVIRONMENT`
- `GOPHER_LOGDEST`
- `GOPHER_LOGLEVEL`
- `GOPHER_SSH_PORT`
- `GOPHER_MESSAGE_FORMAT`

## Task and pipeline environment

Tasks receive a cleaned environment plus pipeline-specific values. Common examples include:

- `GOPHER_ENVIRONMENT`
- `GOPHER_PROTOCOL`
- `GOPHER_USER`
- `GOPHER_CHANNEL`
- `GOPHER_THREAD_ID`
- `GOPHER_MESSAGE_ID`
- `GOPHER_PIPE_NAME`
- `GOPHER_TASK_NAME`
- `GOPHER_START_PROTOCOL`
- `GOPHER_START_CHANNEL`
- `GOPHER_START_THREAD_ID`
- `GOPHER_START_USER`

`GOPHER_ENVIRONMENT` selects environment-specific robot configuration at
startup. It is also exposed to extensions as runtime metadata. The standard v3
authoring convention is to treat `GOPHER_ENVIRONMENT=development` as a local
prove-it mode: plugins that manage host state, cloud resources, firewall rules,
or persistent robot memory should validate inputs and report intended actions
without making those changes.

For task authors, the practical rule is simple: use `GetParameter(...)` for
important values, because it sees explicit parameters and runtime metadata. Use
the process environment for integration with external scripts and tools.
