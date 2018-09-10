#!/bin/bash

# exec.sh - utility task for exec'ing scripts in a repository
# TODO: make this work in containers, remotely, remote containers, etc.

SCRIPT=$1
shift

exec $SCRIPT "$@"
