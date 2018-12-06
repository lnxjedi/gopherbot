#!/bin/bash -e

# exec.sh - utility task for exec'ing scripts in a repository
# TODO: make this work in containers, remotely, remote containers, etc.

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

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
