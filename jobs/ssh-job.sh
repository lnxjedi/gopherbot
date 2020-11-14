#!/bin/bash -e

# ssh-job.sh - simple wrapper job for ssh tasks
# Normal usage is to define multiple jobs with the same path to this script,
# but different values for REMOTEHOST and REMOTETASK (name of task to run).
# Can also call e.g. AddJob ssh-job <host> <task> (args...)

trap_handler()
{
    ERRLINE="$1"
    ERRVAL="$2"
    echo "line ${ERRLINE} exit status: ${ERRVAL}" >&2
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

if [ "$REMOTEHOST" -o "$REMOTETASK" ]
then
    if [ ! \( "$REMOTEHOST" -a "$REMOTETASK" \) ]
    then
        Log "Error" "Only one of REMOTEHOST or REMOTETASK set"
        exit 1
    fi
else
    if [ $# -eq 2 ]
    then
        REMOTEHOST=$1
        REMOTETASK=$2
        shift 2
    else
        Log "Error" "REMOTEHOST and REMOTETASK not set or provided in arguments"
        exit 1
    fi
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
