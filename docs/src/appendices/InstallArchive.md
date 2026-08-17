# A - Gopherbot Installation Archive

> Up-to-date with v2.6

This appendix describes the contents of the **Gopherbot** install archive.

**Files**
* `gopherbot` - the main executable, both a *daemon* and a *command-line interface*
* `cbot.sh` - bash script for setting up, running and developing robots in a container
* `gb-install-links.sh` - trivial utility for creating symlinks to the above

**Directories**
* `conf/` - the default **yaml** configuration files, merged with / overridden by individual robots
    * `conf/robot.yaml` - the primary configuration file for a robot
    * `conf/plugins/` - default configuration for distributed plugins
    * `conf/jobs/` - default configuration for distributed jobs
* `lib/` - API libraries for `bash`, `python` and `ruby`
* `plugins/` - default external script plugins
* `plugins/samples` - sample plugins that show API usage but aren't otherwise very useful
* `tasks/` - a collection of default pipeline task scripts
* `jobs/` - a collection of default jobs for robot management (backup/restore) and CI/CD
* `helpers/` - helper scripts not directly called by the robot
* `resources/` - miscellaneous useful bits for a running robot, also the *Containerfiles* used for publishing the stock containers
* `robot.skel/` - the initial configuration for new robots, analogous to the contents of `/etc/skel`
* `licenses/` - licenses for other packages used by **Gopherbot**, as required
