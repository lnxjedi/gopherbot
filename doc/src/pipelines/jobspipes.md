# Gopherbot Pipelines

Whenever a chat message matches a command plugin, or your robot runs a job, a new pipeline is created; or, more accurately, *three* pipelines are created:

* The **primary** pipeline does all the real work of the job or plugin, and stops on any failed task.
* The **final** pipeline is responsible for cleanup work, and all tasks in this pipeline always run, in reverse of the order added; see the section on [the final pipeline](final.md) for more information.
* The **fail** pipeline runs only after a failure in the primary pipeline.

> NOTE: Jobs, plugins and simple tasks are all instances of a 'task'; the main differences are that 'simple' tasks are meant to be the "worker bees" for job and plugin tasks, which create new pipeline sets. A simple task never creates a new pipeline (though it may add tasks to the current set), it only serves as an element for pipelines created by jobs or plugins.

## Job, Plugin and Task Configuration
Every job, plugin and task needs to be listed in `robot.yaml` (or an included file). See the chapter on [robot configuration](../Configuration.md) for more about this topic.

## Pipeline State
The main state items that connect one element in a pipeline to the next are:
* The **robot** object, generated at the start of the pipeline - this includes a channel where the pipeline is being run, and if started interactively, the user that issued the command.
* The pipeline **environment** - these are environment variables provided to external scripts, or retrievable with `GetParameter(...)` in the **Go** API. Any task in the pipeline can also use the `SetParameter(...)` API call to set/update the environment for tasks that follow in the pipeline; for instance, the `ssh-init` task will set `$SSH_OPTIONS` that should be used with ssh commands later in the pipeline. See the section on [environment variables](../Environment-Variables.md).
* The pipeline **working directory** and it's contents - normally used for software builds, this is generally set early in the pipeline with `SetWorkingDirectory`, with a path relative to `$GOPHER_WORKSPACE`.