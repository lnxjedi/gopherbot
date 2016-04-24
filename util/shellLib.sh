#!/bin/bash
# shellLib.sh - bash plugins should source this with 'source $GOPHER_INSTALLDIR/util/shellLib.sh'

# shellLib.sh needs localdir to suss out the local http port for posting JSON
if [ -z "$GOPHER_LOCALDIR" ]
then
	if [ -d ~/.gopherbot ]
	then
		GOPHER_LOCALDIR=~/.gopherbot
	elif [ -d /etc/gopherbot ]
	then
		GOPHER_LOCALDIR=/etc/gopherbot
	fi
fi
[ -z "$GOPHER_LOCALDIR" ] && { echo "GOPHER_LOCALDIR not found (~/.gopherbot/ or /etc/gopherbot/)" >&2; exit 1; }
[ -z "$GOPHER_INSTALLDIR" ] && { echo "GOPHER_INSTALLDIR not in environment" >&2; exit 1; }

CHANNEL=$1
CHATUSER=$2
COMMAND=$3
PLUGID=$4
shift 4

source $GOPHER_INSTALLDIR/util/shellFuncs.sh
