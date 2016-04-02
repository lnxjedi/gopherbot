#!/bin/bash
# gobot.sh - download/build/install everything bot-related, then run it

usage(){
	cat <<EOF
Usage: gobot.sh (-D)

Download, build and run an instance of gobot-chatops. GOBOT_CONF
must be set to a file containing configuration and credentials.
See the README.md for the details of this file.
EOF
}

errorout(){
	echo "$1" >&2
	exit 1
}

echo "Building / Installing..."
go install
[ $? -ne 0 ] && errorout "Error building, aborting."

[ -z "$GOBOT_CONF" ] && errorout "GOBOT_CONF not set, see README.md"
[ ! -e "$GOBOT_CONF" ] && errorout "File \"$GOBOT_CONF\" (env[GOBOT_CONF]) not found, see README.md"
source $GOBOT_CONF

[ "$1" = "-D" ] && GOBOT_DEBUG=true

[ -z "$GOBOT_CONNECTOR" ] && errorout "GOBOT_CONNECTOR not set, see README.md"

export GOBOT_CONNECTOR
[ -n "$GOBOT_DEBUG" ] && export GOBOT_DEBUG
[ -n "$GOBOT_ALIAS" ] && export GOBOT_ALIAS

case $GOBOT_CONNECTOR in
	slack)
		[ -z "$GOBOT_SLACK_TOKEN" ] && errorout "Error: GOBOT_SLACK_TOKEN not in environment"
		export GOBOT_SLACK_TOKEN
		;;
	*)
		errorout "Unknown/unsupported GOBOT_CONNECTOR: $GOBOT_CONNECTOR"
		;;
esac

echo "Exec'ing bot..."
exec $GOPATH/bin/gobot
