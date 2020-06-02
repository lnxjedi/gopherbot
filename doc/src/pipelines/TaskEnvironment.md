> Note on Parameters and Environment Variables: The **Gopherbot** documentation uses *environment variable* and *parameter* somewhat interchangeably; this is due to configured and set parameters being made available to external scripts as environment variables.

# Task Environment Variables

Each time a task is run, a custom environment is generated for that task. For external tasks such as `ssh-init`, these values are actually set as environment variables for the process. *Go* tasks access these values with the `GetParameter()` API call.

The precedence of environment variables seen by a given task is determined by the algorithm in `bot/run_pipelines.go:getEnvironment()`. The various environment sources are listed here, in order from lowest to highest priority.

## NameSpace Parameters
Various tasks, plugins and jobs can be configured to share parameters by defining `NameSpaces` in `conf/robot.yaml`, and setting the `NameSpace` parameter for a given task to the shared namespace. For instance, the `ssh-init` task and `ssh-admin` plugin both get access to the `BOT_SSH_PHRASE` environment variable via the `ssh` namespace.

## Task Parameters
Individual tasks, plugins and jobs can also have `Parameters` defined. If present, these override any parameters set for a shared namespace.

## Pipeline Parameters
Any time a job is run, the parameters for that job (including those inherited from it's namespace, if any) initialize the environment variables for the pipeline as a whole, and are available to all tasks in the pipeline. Parameters set in the pipeline override task and namespace parameters for any tasks run in the pipeline, allowing specific job parameters to override defaults for the task.

### Plugins and Pipelines
Plugins can also start a new pipeline, but plugin parameters are not automatically added to the pipeline. Plugins can, however, explicity publish parameters to the pipeline with the `SetParameter()` API call.

Plugin tasks can also be added to a pipeline with the `AddCommand()`, `FinalCommand()` and `FailCommand()` API calls. Unlike tasks, plugins only inherit parameters from the pipeline when they are configured with `Privileged: true` in `robot.yaml`.

## Repository Parameters
If a given Job calls the `ExtendNamespace()` API to start a build, the parameters for that repository set in `conf/repositories.yaml` overwrite any values in the current pipeline, which then behave as **Pipeline Parameters** as above.

## SetParameter()
Parameters set with the `SetParameter()` API call overwrite the current value for a pipeline. Parameters set with `SetParameter()` have the highest priority, and will always apply to later tasks in the pipeline.
