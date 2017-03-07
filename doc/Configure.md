## Configuration directories and configuration file precedence

Gopherbot's configuration file loading is designed around installation where credentials and other per-robot variables are configured in the **install directory**, and plugins, additional libraries, and common configuration resides in a **local config directory**, which is likely a git repository. A standard install with a configuration management tool would unzip the archive in /opt/gopherbot and set e.g. the Slack Token in /opt/gopherbot/conf/gopherbot.yaml. Local configuration might be in /usr/local/etc/gopherbot, and external
plugins to load would be defined in /usr/local/etc/gopherbot/conf/gopherbot.yaml.

Configuration file loading simply loads yaml from the **install directory** first, then the **local config directory**, overwriting top-level items in the process.

## Installation Details
### Installdir
When Gopherbot starts, it first looks for it's install directory, containing `lib/` (plugin API libraries for Bash and Ruby). Gopherbot will check for the following directories:
* An installdir passed with the `-i` flag
* `/opt/gopherbot`
* `/usr/local/share/gopherbot`
* `/usr/share/gopherbot`
* `$GOPATH/src/github.com/uva-its/gopherbot`, normally used when installing from source and for plugin development
* The directory where the gopherbot executable resides; this is normally used when running straight from the .zip archive

### Local configuration directory
Gopherbot also looks for a local configuration directory, which must contain at least `conf/`, and optionally `plugins/` for your own plugins. On startup Gopherbot searches for the local config directory in the following locations:
* A config dir passed with the `-c` flag
* `$HOME/.gopherbot` - for plugin development mainly
* `/usr/local/etc/gopherbot`
* `/etc/gopherbot`

## Running Gopherbot as a System Service

**NOTE for Mac Users:** Gopherbot doesn't currently ship with scripts or files for running as a service. PRs welcome.

### Running Gopherbot with systemd on Linux
The contents of `/misc` can be used for setting up the Gopherbot service:
* `gopherbot.service` can be modified and installed in /etc/systemd/system, followed by `systemctl daemon-reload`
* `userdaemon.te` can be used to generate an SELinux policy file allowing Gopherbot to run from the user's home directory (e.g. in $GOPATH)
