# v2.4.3 - DynamoDB Brain support, more HOME updates
This update primary deals with correct operation for robots with non-file brains. It's noteworthy that DynamoDB is the best brain.
* During bootstrapping, the robot was always trying to restore a file-backed (git) brain; now the restore job handles non-file brains gracefully
* During configuration loading, `$HOME` was not being set, causing certain plugins to always be disabled when gems weren't found

# v2.4.2 - Improved IDE UX
This release makes a number of changes to the user experience for the Gopherbot IDE. Notably, it now copies the user's `$HOME/.gitconfig` to the container, so all user git settings apply.

# v2.4.1 - @init jobs
> **NOTE: Potentially breaking change**

This release adds a special '@init' value for scheduled jobs, allowing certain jobs to be run during start-up and reload. The use case for this was to allow decrypting an encrypted `.kube/config` file, but this could also be used for installing python packages and ruby gems.

The **potentially breaking change** is that `$HOME` is now fixed to `$GOPHER_HOME`, the main working directory for the robot. It doesn't make a lot of sense for this to be anything different, or to require this to be set elsewhere (e.g. in the `Dockerfile`).

# v2.4.0 - Gopherbot IDE
This is the first release of a mostly-working Gopherbot IDE, based on [Theia](https://theia-ide.org/). In short:
* Use the `gb-dev-profile` script to generate an IDE profile from a robot's `.env` file - this packages an ssh key and other **git** goodness into a new env file
* Use `gb-start-dev <profile>` to start a local docker container with the profile
* Once the robot starts, tell it to `start-ide` (or `start-theia`), then connect to `http://localhost:3000`

# v2.3.0 - ParameterSets
This release adds `ParameterSets`. Since the `NameSpace` of a task also determines access to long-term memories, tasks can only have a single namespace. ParameterSets are identical in form to NameSpaces, but a given job/task/plugin can list multiple ParameterSets. This allows more flexible sharing of secrets between jobs - especially for secrets like GITHUB_TOKENs, that don't really map well to a namespace. For examples of `ParameterSets` in action, see [Clu](https://github.com/parsley42/clu-gopherbot).

# v2.2.0
* Updated `go.mod` and import paths for (hopefully) 'proper' go module support; since **Gopherbot** is far more an application than a library, this should only matter for indexes like [pkg.go.dev](https://pkg.go.dev)

# v2.1.5 - TOTP Update, Improved help
* Stronger TOTP implementation; generate user codes with the CLI and add encrypted secrets to `conf/plugins/builtin-totp.yaml`
* Make standard "robot, help" contextual (less noisy); "robot, help-all" gives the formerly verbose output
* Export `PYTHONPATH` and `RUBYLIB`, removed ugly env-var references from python & ruby plugins
* Simple tasks now inherit their memory namespace from the pipeline they're added to

# v2.1.4 - Ephemeral Memories
* Allow for ephemeral memories with the git brain; any memories with leading underscores will be ignored by the '.gitignore', allowing developers to prevent frequent git commits for fast changing memories. Also made the .gitignore algorithm more robust.

# v2.1.3
* More container build updates - make all containers more useful for dev env

# v2.1.2
* Update container builds, using GitHub actions for releases and containers

# v2.1.1
* Compile in the dynamodb brain, famously used by Floyd
* Update aws and other deps

# v2.1.0 - Remove Go Modules
Changes since v2.0.2:

The major update for 2.1.0+ is the temporary and possibly permanent removal of modular builds. Support for loadable Go modules was *cool*, but it meant that I couldn't build a single distribution archive installable on all recent Linux distributions due to modular builds using the gcc toolchain linker. Starting with 2.1.0, **Gopherbot** will once again create installable build artifacts.

The other major change is a greater focus on **Slack** as the primary protocol. While the **terminal** connector will continue to be the best way to develop Gopherbot jobs and plugins, the primary build artifacts will be largely Slack-specific.

Other updates/fixes:
* Several fixes to the `autosetup` installer for additional robustness.
