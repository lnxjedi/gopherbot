#!/bin/bash

# localexec.sh - utility task for exec'ing scripts in a repository

SCRIPT=$1
shift

exec $SCRIPT "$@"