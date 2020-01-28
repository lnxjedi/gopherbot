# Installing the `gopherbot` software archive

The latest release, pre-release and beta versions are available for download on the [Github](https://github.com) releases page for [gopherbot](https://github.com/lnxjedi/gopherbot/releases).

1. As **root**, download the `.tar.gz` or `.zip` archive of your choice, and extract the archive in `/opt` to create `/opt/gopherbot`, e.g.:
```shell
[root]# cd /opt
[opt]# wget https://github.com/lnxjedi/gopherbot/releases/download/v2.0.0-snapshot/gopherbot-linux-amd64.tar.gz
[opt]# tar xzf gopherbot-linux-amd64.tar.gz
```
2. (Optional) Also as root, make the `gopherbot` binary **setuid nobody**:
```shell
[opt]# chown nobody gopherbot/gopherbot
[opt]# chmod u+x gopherbot/gopherbot
```

## Archive Contents

**Files**
* `gopherbot` - the main executable, both a *CLI* and *daemon*
* `fetch-robot.sh` - a utility script for retrieving a robot for local development

**Directories**
* `connectors/` - loadable modules for protocol connectors, e.g. *slack*
* `goplugins/` - loadable modules for non-default *Go* plugins
* `brains/` - lodable modules for non-default brain implementations
* `conf/` - the default configuration, overridden by individual robots
* `lib/` - API libraries for `bash`, `python` and `ruby`
* `plugins/` - external script plugins
* `plugins/samples` - sample plugins that show API usage but aren't otherwise useful
* `tasks/` - a collection of pipeline task scripts
* `jobs/` - a collection of jobs for robot management (backup/restore) and CI/CD
* `helpers/` - helper scripts not directly called by the robot
* `resources/` - miscellaneous useful bits for a running robot, also *Dockerfiles*
* `doc/` - the source for the documentation on [github pages](https://lnxjedi.github.io/gopherbot/)
* `robot.skel/` - the initial configuration for new robots
* `licenses/` - licenses for other packages used by **Gopherbot**, as required

# Privilege Separation

**Gopherbot** need never run as root; all of it's privileges derive from the collection of encrypted secrets that a given robot collects. However, given that chat bots tend to accumulate an assortment of 3rd-party command plugins (like the included **Chuck Norris**, hell yeah), **Gopherbot** can be installed *setuid nobody*. Configured this way, the main process may be started by your userid, or a system user such as `robot`, and a given robot will run it's privileged scripts using the `uid` of the parent process. Since almost all toy 3rd-party plugin scripts don't require any privileges, only access to the local API port, **Gopherbot** will run these external scripts with real/effective uid `nobody`. As child processes do not inherit environment, this effectively prevents any potential access to the `GOPHER_ENCRYPTION_KEY`, and any ability to modify the robot's running environment.
