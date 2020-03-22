# Robot Directory Structure

**Gopherbot** robots run in the context of a standard directory structure. The root of this directory structure is the `$GOPHER_HOME` for a given robot, and your robot starts by running the *gopherbot* binary in this directory. This section documents a standard directory structure; though it's possible to modify your particular configuration for different directories, only the standard structure is documented. There are no requirements on this directory except that it needs to be owned and writable by the UID running the *gopherbot* binary; it should not be located under `/opt/gopherbot`, to avoid complicating upgrades.

Note that in the standard directory structure, much of the content is automatically generated when a robot starts. Thanks to the **bootstrap** plugin in the default robot, a fully configured robot can be started with just a `.env` file in an empty directory. In the case of container-based robots, environment variables are provided by the container engine, and all content is generated during bootstrapping.

We'll use [Clu](https://github.com/parsley42/clu-gopherbot) as example:

* `clu/` - Top-level directory, `$GOPHER_HOME`; mostly empty of files and not containing a git repository (`.git/`)
    * `.env` (user provided) - file containing environment variables for the robot, including it's encryption key and git clone url; on start-up the permissions will be forced to `0600`, or start-up will fail - note that this file may be absent in containers
    * `gopherbot` (optional/generated) - convenience symlink to `/opt/gopherbot/gopherbot`
    * `known_hosts` (generated) - used by the robot to record the hosts it connects to over *ssh*
    * `robot.log` (generated) - log file created when the robot is run in **terminal mode**
    * `custom/` - directory for your robot's git repository, containing custom configuration, plugins, jobs and tasks; for *Clu*, this is [https://github.com/parsley42/clu-gopherbot](https://github.com/parsley42/clu-gopherbot)
    * `state/` - for the standard file-backed brain, **state/** contains the robot's encrypted memories in a **state/brain/** directory; this directory is normally linked to the `robot-state` branch of the robot's configuration repository - for *Clu* this is [https://github.com/parsley42/clu-gopherbot/tree/robot-state](https://github.com/parsley42/clu-gopherbot/tree/robot-state)
    * `history/` - the standard robot keeps job / plugin logs here
    * `workspace/` - default location for the robot's workspace, where repositories are cloned, etc.

## The `custom/` directory

The `custom/` directory is essentially *your robot*, and is generally used differently for development and deployment.

### During Development

When developing jobs, tasks and plugins for your robot, you may provide the `.env` file, manually clone your robot's repository to `custom/`, and treat `state/` as disposable. A fairly standard workflow goes like this:
1. Copy the robot's `.env` to an empty directory, with `GOPHER_PROTOCOL` commented out; when you start a standard robot this way, the bootstrap plugin will clone the robot to `custom/` and start in **terminal connector** mode
   1. It's also possible, but generally unecessary, to manually clone
1. Use the **terminal** connector, configured to mirror your team chat environment, for developing extensions for your robot
1. From the `custom/` directory, manually push changes with appropriate commit messages
1. Send an administrator `update` command to your production robot to pull down the latest changes and reload
1. Remove everything but `.env` until the next time you work on the robot

For more information on developing, see the chapter on [Developing Extensions](../botprogramming.md).

### Deployment to Production

Production robots normally clone `custom/` the first time they start on a new VM or container, and update by an administrator `update` command.

> Production robots can also be configured to automatically update whenever the master branch updates. **Floyd**, for example, does this. See the `Triggers:` section in Floyd's [updatecfg.yaml file](https://github.com/parsley42/floyd-gopherbot/blob/master/conf/jobs/updatecfg.yaml).