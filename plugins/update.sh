#!/bin/bash -e

# update.sh - a Bash plugin that triggers the updatecfg job.
# Requires enabling the updatecfg job, and tasks for ssh-init and git.

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

COMMAND=$1
shift

configure(){
  cat <<"EOF"
RequireAdmin: true
ElevatedCommands: [ 'update' ]
AllowDirect: true
Help:
- Keywords: [ "config", "configuration", "update" ]
  Helptext: [ "(bot), update (configuration) - perform a 'git pull' in the configuration directory and reload" ]
CommandMatchers:
- Command: "update"
  Regex: '(?i:update(?: config(?:uration)?)?)'
EOF
}

case "$COMMAND" in
	"configure")
		configure
		;;
  "update")
    Say "Ok, I'll trigger the 'updatecfg' job to issue a git pull and reload configuration..."
    AddTask updatecfg
    FailTask notify $GOPHER_USER "Update failed, check history for the 'updatecfg' job"
    ;;
esac
