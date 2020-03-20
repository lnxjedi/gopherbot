# Robot Directory Structure

**Gopherbot** robots run in the context of a standard directory structure. The root of this directory structure is the `$GOPHER_HOME` for a given robot, and your robot starts by running the *gopherbot* binary in this directory. This section documents a standard directory structure; though it's possible to modify your particular configuration for different directories, only the standard structure is documented. There are no requirements on this directory except that it needs to be owned and writable by the UID running the *gopherbot* binary; it should not be located under `/opt/gopherbot`, to avoid complicating upgrades.

We'll use [Clu](https://github.com/parsley42/clu-gopherbot) as example:

* `clu/` - Top-level directory, `$GOPHER_HOME`; mostly empty of files and not containing a git repository (`.git/`)
    * `gopherbot` - optional convenience symlink to `/opt/gopherbot/gopherbot`
    * `.env` - file containing environment variables for the robot, including it's encryption key and git clone url; on start-up the permissions will be forced to `0600`, or start-up will fail
    * `known_hosts` - used by the robot to record the hosts it connects to over *ssh*
    * `robot.log` - log file created when the robot is run in **terminal mode**
    * `custom/` - directory for your robot's git repository, containing custom configuration, plugins, jobs and tasks; for *Clu*, this is [https://github.com/parsley42/clu-gopherbot](https://github.com/parsley42/clu-gopherbot)
    * `state/` - for the standard file-backed brain, **state/** contains the robot's encrypted memories in a **state/brain/** directory; this directory is normally linked to the `robot-state` branch of the robot's configuration repository - for *Clu* this is [https://github.com/parsley42/clu-gopherbot/tree/robot-state](https://github.com/parsley42/clu-gopherbot/tree/robot-state)
    * `history/` - the standard robot keeps job / plugin logs here
    * `workspace/` - default location for the robot's workspace, where repositories are cloned, etc.