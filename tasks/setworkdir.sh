#!/bin/bash -e

# tasks/setworkdir.sh - update working directory during pipeline

source $GOPHER_INSTALLDIR/lib/gopherbot_v1.sh

SetWorkingDirectory $1
