**NOTE on Parameters and Environment Variables**: The **Gopherbot** documentation uses *environment variable* and *parameter* somewhat interchangeably; this is due to configured parameters being made available to external scripts as environment variables.

# Task Environment Variables

Each time a task is run, a custom environment is generated for that task. For external tasks such as `ssh-init`, these values are actually set as environment variables for the process. *Go* tasks access these values with the `GetParameter()` API call.

The precedence of environment variables seen by a given task is determined by the algorithm in `bot/runtasks.go:getEnvironment()`. The various environment sources are listed here, in order from least to highest priority.

## NameSpace Parameters

Various tasks, plugins and jobs can be configured to share parameters by defining `NameSpaces` in `conf/gopherbot.yaml`, and setting the `NameSpace` parameter for a given task to the shared namespace. For instance, the `ssh-init` task and `ssh-admin` plugin both get access to the `BOT_SSH_PHRASE` environment variable via the `ssh` namespace.

## Task Parameters
Individual tasks, plugins and jobs can also have `Parameters` defined. If present, these override any parameters set for a shared namespace.

## Pipeline Parameters
The least-specific environment variables are those set for the pipeline. Any time a job or plugin is run, the parameters for that job or plugin initialize the environment variables for the pipeline as a whole, and are available to all tasks in the pipeline.

### Repository Parameters
If a given Job calls the `ExtendNamespace()` API to start a build, the parameters for that repository set in `conf/repositories.yaml` overwrite any values in the current pipeline.

### SetParameter()
Parameters set with the `SetParameter()` API call overwrite the current value for a pipeline; thus, it makes little sense to call `SetParameter()` before `ExtendNamespace()`.
