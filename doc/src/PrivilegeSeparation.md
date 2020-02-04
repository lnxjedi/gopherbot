## Privilege Separation

One of **Gopherbot**'s primary tasks is executing external scripts, which may in some cases be third-party plugins. To harden the environment in which these scripts are run, the `gopherbot` binary can be installed setuid to a non-privileged user such as `nobody`. The following rules apply to external scripts as privileged/unprivileged:

* Unless a plugin is configured with `Privileged: true` in `robot.yaml`, the plugin will run as the unprivileged user
* Jobs will always run privileged
* Only a privileged plugin will be able to add a job to the current pipeline
* If an external plugin command is added to a pipeline with `AddCommand`, that plugin will still run unprivileged
* Tasks will run with the privileges of the current pipeline
