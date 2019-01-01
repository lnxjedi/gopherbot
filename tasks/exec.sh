#!/bin/bash -e

# exec.sh - utility task for exec'ing scripts in a repository
# TODO: make this work in containers, remotely, remote containers, etc.

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

for ARG in "$@"
do
    if [[ $ARG = *=* ]]
    then
        VAR=${ARG%%=*}
        VALUE=${ARG#*=}
        export $VAR="$VALUE"
        shift
    fi
done

SCRIPT=$1
shift

if [[ $SCRIPT != /* && $SCRIPT == */* ]]
then
    if [ ! -e $SCRIPT ]
    then
        Log "Warn" "Script not found: $SCRIPT, ignoring"
        exit 0
    fi
fi

exec $SCRIPT "$@"
