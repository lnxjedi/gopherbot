#!/bin/bash
# gobot.sh - download/build/install everything bot-related, then run it

usage(){
	cat <<EOF
Usage: gopherbot.sh (-h) (-f)

Run an instance of gopherbot. GOPHER_LOCALDIR should point
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
[ -z "$GOPHER_INSTALLDIR" ] && export GOPHER_INSTALLDIR=$EXECPATH

[ ! -d "$GOPHER_LOCALDIR" ] && errorout "GOPHER_LOCALDIR not set to a directory, see README.md"
[ ! -e "$GOPHER_LOCALDIR/conf/gopherbot.json" ] && errorout "Couldn't find gopherbot.json in $GOPHER_LOCALDIR/conf/"

echo "Exec'ing bot..."
if [ -n "$1" ]
then
	$EXECPATH/gopherbot
else
	$EXECPATH/gopherbot 2> /tmp/gopherbot.log &
fi
