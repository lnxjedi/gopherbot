# Custom Configuration Directory

Version 2's configuration layout is heavily oriented towards an all-in-one robot directory, `$GOPHER_HOME`, with brain, job logs, workspaces, and others as subdirectories. It standardizes the location of the custom configuration directory (repository) to `$(pwd)/custom` (a.k.a. `$GOPHER_HOME/custom`). While setting the location with `-c <dir>` or via the `GOPHER_CONFIGDIR` environment variable are still supported, existing installations should locate custom configuration in a directory named `custom/`, with the robot started from the parent directory. See the [Installation Overview](../InstallOverview.md) for more information on the standard directory structure.

If your configuration is already in a repository, and the previous updates have already been made, then you can probably just run `/opt/gopherbot/gopherbot` from an empty directory and use the new setup plugin to bootstrap your robot.

## Optional rename of `gopherbot.yaml`

Version 2 defaults to `robot.yaml` for the name of the main configuration file for your robot. For backwards-compatibility, gopherbot will fall back to `gopherbot.yaml`.
