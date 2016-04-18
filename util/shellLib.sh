#!/bin/bash

if [ $# -lt 3 ]
then
	echo "Usage: $0 <channel> <user> <command> (<args>...)"
	exit 1
fi

[ $# -lt 2 ] && { echo "Usage: sendusermsg.sh <user> message" >&2; exit 1; }

CHANNEL=$1
USER=$2
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
	echo "$JSON" | curl -X POST -d @- http://localhost:$GOBOT_LOCALPORT/json
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
	echo "$JSON" | curl -X POST -d @- http://localhost:$GOBOT_LOCALPORT/json
}

