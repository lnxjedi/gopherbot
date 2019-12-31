#!/bin/bash -e

# update.sh - a Bash plugin that triggers the updatecfg job.
# Requires enabling the updatecfg job, and tasks for ssh-init and git.

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

COMMAND=$1
shift

configure(){
  cat <<"EOF"
RequireAdmin: true
AllowDirect: true
Help:
- Keywords: [ "config", "configuration", "update" ]
  Helptext: [ "(bot), update (configuration) - perform a git clone/pull of custom configuration and reload" ]
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
    AddJob updatecfg
    AddTask say "... done"
    ;;
esac
