#!/bin/bash -e

# exec.sh - utility task for exec'ing scripts in a repository
# TODO: make this work in containers, remotely, remote containers, etc.

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

SCRIPT=$1
shift

exec $SCRIPT "$@"
