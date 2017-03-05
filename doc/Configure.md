## Installation Details
### Installdir
When Gopherbot starts, it first looks for it's install directory, containing at least `lib/` (plugin API libraries for Bash and Ruby), and `plugins/` (default/sample plugins shipped with Gopherbot). Gopherbot will check for the following directories:
* An installdir passed with the `-i` flag
* The location specified in the `$GOPHER_INSTALLDIR` environment variable, if set
* `/usr/local/share/gopherbot`
* `/usr/share/gopherbot`
* `$GOPATH/src/github.com/uva-its/gopherbot`, normally used when installing from source
* The directory where the gopherbot executable resides; this is normally used when running straight from the .zip archive

You can copy `lib/` and `plugins/` to one of the directories listed above (mainly for packaging/deployment), but for development and testing you can run gopherbot from the expanded archive directory and it'll look for it's install files there.

### Local configuration directory
Gopherbot also looks for a local configuration directory, which must contain at least `conf/`, and optionally `plugins/` for your own plugins. You can copy the `example.gopherbot/` directory to one of the following locations that Gopherbot searches for local configuration:
* A config dir passed with the `-c` flag
* The location specified in the `$GOPHER_CONFIGDIR` environment variable, if set
* `$HOME/.gopherbot`
* `/usr/local/etc/gopherbot`
* `/etc/gopherbot`

You'll need to edit the example config and at least set a correct SlackToken (obtainable from https://yourteam.slack.com/services/new/bot), but you should probably add your username to AdminUsers to enable e.g. the `reload` command.
