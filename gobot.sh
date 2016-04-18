#!/bin/bash
# gobot.sh - download/build/install everything bot-related, then run it

usage(){
	cat <<EOF
Usage: gobot.sh

Build and run an instance of gobot. GOBOT_CONFIGDIR should point
to a directory holding gobot.json and any plugin configuration files.
EOF
}

errorout(){
	echo "$1" >&2
	exit 1
}

echo "Building / Installing..."
go install
[ $? -ne 0 ] && errorout "Error building, aborting."

EXECPATH=$(dirname `readlink -f $0`)
echo "EXECPATH:$EXECPATH"
export GOBOT_SHELLLIB="$EXECPATH/util/shellLib.sh"

[ ! -d "$GOBOT_CONFIGDIR" ] && errorout "GOBOT_CONFIGDIR not set to a directory, see README.md"
[ ! -e "$GOBOT_CONFIGDIR/gobot.json" ] && errorout "Couldn't find gobot.json in $GOBOT_CONFIGDIR"

echo "Exec'ing bot..."
exec $GOPATH/bin/gobot
