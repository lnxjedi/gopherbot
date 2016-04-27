#!/bin/bash
# shellLib.sh - bash plugins should source this with 'source $GOPHER_INSTALLDIR/util/shellLib.sh'

GB_CHANNEL=$1
GB_USER=$2
GB_PLUGID=$3
shift 3
# Now $1 is the command

source $GOPHER_INSTALLDIR/util/shellFuncs.sh
