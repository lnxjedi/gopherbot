#!/bin/bash
# gobot.sh - download/build/install everything bot-related, then run it

usage(){
	cat <<EOF
Usage: gopherbot.sh

Build and run an instance of gopherbot. GOPHER_LOCALDIR should point
to a directory holding gopherbot.json and any plugin configuration files.
EOF
}

errorout(){
	echo "$1" >&2
	exit 1
}

echo "Building gobot-chatops..."
go build
[ $? -ne 0 ] && errorout "Error building, aborting."

EXECPATH=$(dirname `readlink -f $0`)
[ -z "$GOPHER_INSTALLDIR" ] && export GOPHER_INSTALLDIR=$EXECPATH

[ ! -d "$GOPHER_LOCALDIR" ] && errorout "GOPHER_LOCALDIR not set to a directory, see README.md"
[ ! -e "$GOPHER_LOCALDIR/conf/gopherbot.json" ] && errorout "Couldn't find gopherbot.json in $GOPHER_LOCALDIR"
export GOPHER_SHELLLIB="$GOPHER_INSTALLDIR/util/shellLib.sh"

echo "Exec'ing bot..."
$EXECPATH/gopherbot 2> /tmp/gopherbot.log &
