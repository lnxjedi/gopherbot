#!/bin/bash -e

# restore.sh - restore the robot's state from git

trap_handler()
{
    ERRLINE="$1"
    ERRVAL="$2"
    echo "line ${ERRLINE} exit status: ${ERRVAL}"
    # The script should usually exit on error
    exit $ERRVAL
}
trap 'trap_handler ${LINENO} $?' ERR

for REQUIRED in git jq ssh
do
    if ! which $REQUIRED >/dev/null 2>&1
    then
        echo "Required '$REQUIRED' not found in \$PATH"
        exit 1
    fi
done

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

if [ $# -eq 2 ]
then
    REMOTEHOST=$1
    REMOTETASK=$2
    shift 2
fi

if [ ! "$REMOTEHOST" ]
then
    Log "Error" "REMOTEHOST not provided in parameters or arguments"
    exit 1
fi

if [ ! "$REMOTETASK" ]
then
    Log "Error" "REMOTETASK not provided in parameters or arguments"
    exit 1
fi

AddTask ssh-init
AddTask ssh-scan $REMOTEHOST
AddTask $REMOTETASK "$@"
exit 0