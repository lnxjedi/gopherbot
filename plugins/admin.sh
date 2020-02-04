#!/bin/bash -e

# admin.sh - a bash plugin that triggers management jobs like save, update, etc.

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

COMMAND=$1
shift

case "$COMMAND" in
	"configure")
		exit 0
		;;
  "update")
    Say "Ok, I'll trigger the 'updatecfg' job to issue a git pull and reload configuration..."
    AddJob updatecfg
    FailTask say "Job failed!"
    AddTask say "... done"
    ;;
  "save")
    Say "Ok, I'll push my configuration..."
    AddJob save
    FailTask say "Job failed!"
    AddTask say "... done"
    ;;
  "backup")
    Say "Ok, I'll start the backup job to push my state..."
    AddJob backup
    FailTask say "Job failed!"
    AddTask say "... done"
    ;;
  "restore")
    Say "Ok, I'll start a restore of my state from the remote repository..."
    AddJob restore "$1"
    FailTask say "Job failed!"
    AddTask say "... done"
    ;;
esac
