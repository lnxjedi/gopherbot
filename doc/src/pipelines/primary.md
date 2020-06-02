# The Primary Pipeline
For those with previous experience with CI/CD and pipelines, the **primary** pipeline is analogous to the pipelines you've worked with before. Each task in the pipeline runs until a task fails or the pipeline completes successfully.

Unlike statically configured pipelines, once a job or plugin starts a pipeline, the `AddTask` and `AddCommand` API calls are used to add tasks to this pipeline, and can vary, for instance, based on environment variables such as `$GOPHERCI_BRANCH` in a software build.

In the case of most plugins, there is only a single task in the primary pipeline, the plugin itself. In the case of most jobs, the job task normally adds tasks to each of the pipelines and then exits successfully, at which point the next task in the pipeline runs.

## Initializing the Environment
There are a few differences between job and plugin pipelines, one of which is the initial environment. For jobs, any parameters set for the job are propagated to the environment for the entire pipeline; plugins must use the `SetParameter(...)` API call to propagate values. Environment variables are more thoroughly explained in the section on [task environment variables](TaskEnvironment.md).

## Populating the Pipeline Set
During the execution of the primary pipeline, the `Add*`, `Final*` and `Fail*` family of API calls can be used to populate the pipeline set; these calls will fail if used in the final and fail pipelines:
* `AddTask`, `FinalTask` and `FailTask` add simple tasks to the pipeline set.
* `AddJob` adds another job to the pipeline, optionally with arguments; note, however, that this creates an entirely new child pipeline, with a new environment not inherited from the parent. If the child job pipeline fails, the parent pipeline also fails.
* `AddCommand`, `FinalCommand` and `FailCommand` are mostly special-purpose and generally little used, they add plugin commands to the respective pipelines in the set. One use is in wrapping existing plugins with shortcut commands in another plugin - for instance, when I say to my robot "dinner?", it calls `AddCommand "lists" "pick a random item from the dinner meals list"`; see [Floyd's util plugin](https://github.com/parsley42/floyd-gopherbot/blob/master/plugins/util.sh). Adding plugins to the pipeline does *not* create a new child pipeline.

## Order of Task Execution
The normal case is having an initial job or plugin task that fully populates the primary, final and fail pipelines. There are some tasks, however, that add one or more additional tasks to the pipeline set. The final and fail pipelines just grow in length, but the behavior of the primary pipeline is special:
* If tasks are added in the last task of the pipeline, the added tasks are just appended and the pipeline runs them as normal.
* If the tasks are added BEFORE the final task in the pipeline, **all** of the added tasks run before the next task in the original pipeline run. As an example:

**Job A**:
```
AddTask A
AddTask B
AddTask C
AddTask D
```

**Task C**:
```
AddTask E
AddTask F
```

Assuming none of the other tasks add additional tasks, they will run in the order: A, B, C, E, F, D. This allows for slightly more complicated tasks that may, for instance, have several sub-steps in setting up the working directory, and need to all complete before the next task in the job proceeds.

In the case of **GopherCI**, the `localbuild` job adds (among others) the following tasks:
* "startbuild" - a task to provide information about the build that is starting
* "run-pipeline" - a task that runs a pipeline specified in the repository
* "finishbuild" - a task to report on how the build ended

The contents of `<repository>/.gopherci/pipeline.sh` correspond most closely to the contents of e.g. `<repository>/.travis.yml` - they define the build, test and deploy tasks for the repository-specific pipeline. The task execution order for the primary pipeline insures that all the tasks in the repository pipeline are run before any other tasks added by the build job.

> NOTE: "finishbuild" is actually added to the final pipeline so it always runs, however the example still holds.