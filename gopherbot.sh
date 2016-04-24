#!/bin/bash
# gopherbot.sh - run the gopherbot

usage(){
	cat <<EOF
Usage: gopherbot.sh (-h) (-f)

Run an instance of gopherbot. GOPHER_INSTALLDIR should point
to a directory holding gopherbot.json and any plugin configuration files.
	-f run in foreground
	-h print this message
EOF
	exit 0
}

[ "$1" = "-h" ] && usage

errorout(){
	echo "$1" >&2
	exit 1
}

EXECPATH=$(dirname `readlink -f $0`)
[ -z "$GOPHER_INSTALLDIR" ] && GOPHER_INSTALLDIR=$EXECPATH
[ -e "$GOPHER_INSTALLDIR/gopherbot" ] && echo "WARNING: found $EXECPATH/gopherbot, you might not be running the latest build. Use build.sh" >&2

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
[ -z "$GOPHER_LOCALDIR" ] && errorout "GOPHER_LOCALDIR not found (~/.gopherbot/ or /etc/gopherbot/)"

[ ! -d "$GOPHER_INSTALLDIR" ] && errorout "GOPHER_INSTALLDIR not set to a directory, see README.md"
[ ! -e "$GOPHER_INSTALLDIR/conf/gopherbot.json" ] && errorout "Couldn't find gopherbot.json in $GOPHER_INSTALLDIR/conf/"

export GOPHER_INSTALLDIR GOPHER_LOCALDIR

echo "Exec'ing bot..."
if [ -n "$1" ]
then
	$EXECPATH/robot
else
	$EXECPATH/robot 2> /tmp/gopherbot.log &
fi
