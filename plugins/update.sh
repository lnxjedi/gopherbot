#!/bin/bash -e

# update.sh - a Bash plugin allowing the robot to
# update it's Configuration Directory using git.
# It's up to the bot admin to install an ssh keypair for the
# bot in the $HOME/.ssh directory that has at least
# read access to the git repository. (normally a deploy key)
# NOTE: this plugin requires external tasks for git, ssh, and
# ssh-agent; see gopherbot.yaml.sample.

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

COMMAND=$1
shift

configure(){
  cat <<"EOF"
RequireAdmin: true
ElevatedCommands: [ 'update' ]
AllowDirect: true
HistoryLogs: 7
Help:
- Keywords: [ "config", "configuration", "update" ]
  Helptext: [ "(bot), update configuration - perform a 'git pull' in the configuration directory" ]
CommandMatchers:
- Command: "update"
  Regex: '(?i:update config(?:uration)?)'
EOF
}

case "$COMMAND" in
	"configure")
		configure
		;;
    "update")
        Say "Ok, I'll issue a git pull and reload configuration..."
        AddTask ssh-init
        AddTask git pull
        AddTask builtInadmin reload
        ;;
esac
