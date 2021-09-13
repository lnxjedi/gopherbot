#!/bin/bash -e

# theia.sh - a bash plugin that triggers the theia job.

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
