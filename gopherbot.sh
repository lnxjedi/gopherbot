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

export GOPHER_LOCALDIR

EXECPATH=$(dirname $0)
echo "Exec'ing bot..."
if [ -n "$1" ]
then
	$EXECPATH/robot
else
	$EXECPATH/robot 2> /tmp/gopherbot.log &
fi
