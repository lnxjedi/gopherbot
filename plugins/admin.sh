#!/bin/bash -e

# admin.sh - a bash plugin for legacy admin workflows

COMMAND=$1
shift

[ "$COMMAND" = "configure" ] && exit 0

trap_handler()
{
  ERRLINE="$1"
  ERRVAL="$2"
  echo "line ${ERRLINE} exit status: ${ERRVAL}" >&2
  # The script should usually exit on error
  exit $ERRVAL
}
trap 'trap_handler ${LINENO} $?' ERR

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

FailTask tail-log

case "$COMMAND" in
  "update")
    Say "Ok, I'll trigger the 'updatecfg' job to issue a git pull and reload configuration..."
    AddJob updatecfg
    FailTask say "Job failed!"
    AddTask say "... done"
    ;;
  "branch")
    BRANCH="$1"
    AddJob go-switchbranch "$BRANCH"
    FailTask say "Error switching branches - does '$BRANCH' exist?"
    FailTask tail-log
    AddTask send-message "... switched to branch '$BRANCH'"
    ;;
  "theia")
    if [ ! -e "/usr/local/theia/src-gen/backend/main.js" ]
    then
      Say "Theia installation not found. Wrong container?"
      exit 0
    fi
    Say "Ok, I'll start the Theia Gopherbot IDE..."
    AddJob theia
    FailTask say "Starting theia failed! (are you using the gopherbot-theia image?)"
    AddTask say "... Theia finished"
    ;;
esac
