___

**ssh-git-helper** - *privileged*

Usage:
- `AddTask ssh-git-helper addhostkeys <encoded-host-keys>` - create a `known_hosts` file from the encoded keys
- `AddTask ssh-git-helper scan <host>` - fallback sub-command to scan a remote host for the host keys
- `AddTask ssh-git-helper loadhostkeys <repo uri>` - for github, bitbucket, retrieve host keys from a known https API endpoint
- `AddTask ssh-git-helper publishenv` - set `SSH_OPTIONS` and `GIT_SSH_COMMAND` for external binary use
- `AddTask ssh-git-helper delete` - delete the temporary hostkeys file; automatically added as a FinalTask

The `addhostkeys`, `loadhostkeys` and `scan` sub-commands all create a `known_hosts` file that can be used with `git-command` and `ssh-agent` for working with git repositories authenticated with ssh. Each of these sub-commands also automatically adds a `FinalTask ssh-git-helper delete` to remove the temporary file. By using `publishenv`, the `SSH_OPTIONS` and `GIT_SSH_COMMAND` parameters (environment variables) are exported to the pipeline for use by e.g. bash-based tasks using the system ssh and/or git binaries.

Note that `<encoded-host-keys>` is a single-line string encoded similarly to `GOPHER_DEPLOY_KEY`; all spaces are replaced with '_', and newlines are replaced with ':'; basically, `tr ' \n' '_:'`.
