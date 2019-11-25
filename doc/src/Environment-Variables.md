# Gopherbot Environment Variables

**Gopherbot** makes extensive use of environment variables, both for configuring the robot and plugins, and for providing parameters to external scripts.

## External Script Environment

**Gopherbot** always scrubs the environment when executing tasks, so environment variables set on execution are not automatically passed to child processes. The only environment variables that are passed through from original execution are:
* `HOME`
* `HOSTNAME`
* `LANG`
* `PATH`
* `USER`

In addition to the above passed-through environment vars, **Gopherbot** supplies the following environment variables to external scripts:
* `GOPHER_INSTALLDIR` - the location of the gopherbot install, normally `/opt/gopherbot`
* `GOPHER_CONFIGDIR` - the location of the custom configuration directory, `$(pwd)/custom`

## Job Environment Variables

The following environment variables are supplied whenever a job is run:
* `GOPHER_JOB_NAME` - the name of the running job
* `GOPHER_TASK_NAME` - the name of the running task
* `GOPHER_NAMESPACE_EXTENDED` - the extended namespace (minus the branch), if any
* `GOPHER_RUN_INDEX` - the run number of the job
* `GOPHER_WORKSPACE` - the initial working directory when jobs are run
* `GOPHER_PIPELINE_TYPE` - the event type that started the current pipeline, one of:
    * `plugCommand` - direct robot command, not `run job ...`
    * `plugMessage` - ambient message matched
    * `catchAll` - catchall plugin ran
    * `jobTrigger` - triggered by a JobTrigger
    * `scheduled` - started by a ScheduledTask
    * `jobCmd` - started from `run job ...` command

In addition, the `localbuild` GopherCI builder sets the following environment variables that can be used to modify pipelines:
* `GOPHERCI_REPO` - the repository being built
* `GOPHERCI_BRANCH` - the branch being built
* `GOPHERCI_DEPBUILD` - set to "true" if the build was triggered by a dependency
* `GOPHERCI_DEPREPO` - the updated repository that triggered this build
* `GOPHERCI_DEPBRANCH` - the updated branch
