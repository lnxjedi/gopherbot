#!/bin/bash

if [ $# -lt 3 ]
then
	echo "Usage: $0 <channel> <user> <command> (<args>...)"
	exit 1
fi

[ $# -lt 2 ] && { echo "Usage: sendusermsg.sh <user> message" >&2; exit 1; }

CHANNEL=$1
CHATUSER=$2
COMMAND=$3
shift 3

PORTMATCH=$(grep LocalPort "$GOBOT_CONFIGDIR/gobot.json")
PORTMATCH=${PORTMATCH%\",}
PORTMATCH=${PORTMATCH##*\"}
GOBOT_LOCALPORT=$PORTMATCH

sendUserMessage(){
	CHATUSER=$1
	shift
	MESSAGE="$*"

	JSON=$(cat <<EOF
{
	"Command": "SendUserMessage",
	"CmdArgs": {
		"User": "$CHATUSER",
		"Message": "$MESSAGE"
	}
}
EOF
)
	echo "$JSON" | curl -X POST -d @- http://localhost:$GOBOT_LOCALPORT/json 2>/dev/null
}

sendChannelMessage(){
	CHANNEL=$1
	shift
	MESSAGE="$*"

JSON=$(cat <<EOF
{
	"Command": "SendChannelMessage",
	"CmdArgs": {
		"Channel": "$CHANNEL",
		"Message": "$MESSAGE"
	}
}
EOF
)
	echo "$JSON" | curl -X POST -d @- http://localhost:$GOBOT_LOCALPORT/json 2>/dev/null
}

# Convenience functions so that copies of this logic don't wind up in a bunch of plugins
say(){
	if [ -n "$CHANNEL" ]
	then
		sendChannelMessage "$CHANNEL" "$*"
	else
		sendUserMessage "$CHATUSER" "$*"
	fi
}

reply(){
	if [ -n "$CHANNEL" ]
	then
		sendChannelMessage "$CHANNEL" "@$CHATUSER:" "$*"
	else
		sendUserMessage "$CHATUSER" "$*"
	fi
}
