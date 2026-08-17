___

**ssh-agent** - *privileged*

Usage:
- `AddTask ssh-agent start <path/to/key> <timeout>` - requires `BOT_SSH_PHRASE` in parameters
- `AddTask ssh-agent stop` - uses `_SSH_AGENT_HANDLE`, automatically added as `FinalTask` by start, deploy
- `AddTask ssh-agent deploy <timeout>` - requires `GOPHER_DEPLOY_KEY` in parameters

The `ssh-agent` task uses Go ssh-agent libraries to start a new Goroutine loaded with a deployment key, or regular encrypted key. The `start` and `deploy` sub-commands automatically add a `FinalTask ssh-agent stop` to the pipeline to ensure cleanup. This task can be used with `ssh-git-helper` and `git-command` to work with git repositories.

The `start` and `deploy` sub-commands also create a socket in `$GOPHER_HOME` and sets the `SSH_AUTH_SOCK` parameter in the pipeline, so e.g. a "bash" task could use native ssh commands.

See also:
- plugins/samples/bootstrap.py - previous bootstrapping pipeline
