#!/bin/bash -e

# admin.sh - a bash plugin that triggers management jobs like save, update, etc.

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

for REQUIRED in git jq ssh
do
    if ! which $REQUIRED >/dev/null 2>&1
    then
        echo "Required '$REQUIRED' not found in \$PATH"
        exit 1
    fi
done

case "$COMMAND" in
  "update")
    Say "Ok, I'll trigger the 'updatecfg' job to issue a git pull and reload configuration..."
    AddJob updatecfg
    FailTask say "Job failed!"
    AddTask say "... done"
    ;;
  "branch")
    BRANCH="$1"
    AddJob changebranch "$BRANCH"
    FailTask say "Error switching branches - does '$BRANCH' exist?"
    AddTask say "... switched to branch '$BRANCH'"
    ;;
  "save")
    Say "Ok, I'll push my configuration..."
    AddJob save
    FailTask say "Job failed!"
    AddTask say "... done"
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
