#!/bin/bash
# shellLib.sh - bash plugins should source this with 'source $GOPHER_INSTALLDIR/util/shellLib.sh'
[ -z "$GOPHER_LOCALDIR" ] && { echo "GOPHER_LOCALDIR not set" >&2; exit 1; }
[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not set" >&2; exit 1; }

if [ $# -lt 3 ]
then
	echo "Usage: $0 <channel> <user> <command> (<args>...)"
	exit 1
fi

CHANNEL=$1
CHATUSER=$2
COMMAND=$3
PLUGID=$4
shift 4

source $GOPHER_INSTALLDIR/util/shellFuncs.sh
