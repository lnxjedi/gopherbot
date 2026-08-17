## Privilege Separation

One of **Gopherbot**'s primary tasks is executing external scripts, which may in some cases be third-party plugins. To harden the environment in which these scripts are run, the `gopherbot` binary can be installed setuid to a non-privileged user such as `nobody`. The following rules apply to external scripts as privileged/unprivileged:

* Plugins default to unprivileged unless configured with `Privileged: true` in `robot.yaml`.
* Jobs default to privileged unless configured otherwise.
* A plugin or job configured with `Privileged: true` starts a privileged pipeline.
* A task configured with `Privileged: true` requires the current pipeline to already be privileged; `AddTask`, `FinalTask`, and `FailTask` do not elevate an unprivileged pipeline.
* If an external plugin command is added to a pipeline with `AddCommand`, that plugin still runs according to the current pipeline and plugin privilege rules.
* Tasks run with the privileges of the current pipeline. For example, a task that runs `kubectl` with a robot-managed kubeconfig should be marked `Privileged: true`, and only a privileged job or plugin should add it.
