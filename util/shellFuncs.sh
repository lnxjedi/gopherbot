PORTMATCH=$(grep LocalPort "$GOPHER_LOCALDIR/conf/gopherbot.json")
PORTMATCH=${PORTMATCH%\",}
PORTMATCH=${PORTMATCH##*\"}
GOPHER_LOCALPORT=$PORTMATCH

encode(){
	local MESSAGE
	MESSAGE=$(echo "$@" | base64)
	MESSAGE=$(echo "base64:$MESSAGE")
	echo "$MESSAGE"
}

sendUserMessage(){
	local CHATUSER CHANNEL
	GOPHER_MESSAGE_FORMAT=${GOPHER_MESSAGE_FORMAT:-variable}
	CHATUSER=$1
	shift
	MESSAGE="$*"
	MESSAGE=$(encode "$MESSAGE")

	JSON=$(cat <<EOF
{
	"Command": "SendUserMessage",
	"CmdArgs": {
		"User": "$CHATUSER",
		"Format": "$GOPHER_MESSAGE_FORMAT",
		"Message": "$MESSAGE"
	}
}
EOF
)
	echo "$JSON" | curl -X POST -d @- http://localhost:$GOPHER_LOCALPORT/json 2>/dev/null
}

sendUserChannelMessage(){
	local CHATUSER CHANNEL
	GOPHER_MESSAGE_FORMAT=${GOPHER_MESSAGE_FORMAT:-variable}
	CHATUSER=$1
	CHANNEL=$2
	shift 2
	MESSAGE="$*"
	MESSAGE=$(encode "$MESSAGE")

	JSON=$(cat <<EOF
{
	"Command": "SendUserChannelMessage",
	"CmdArgs": {
		"User": "$CHATUSER",
		"Channel": "$CHANNEL",
		"Format": "$GOPHER_MESSAGE_FORMAT",
		"Message": "$MESSAGE"
	}
}
EOF
)
	echo "$JSON" | curl -X POST -d @- http://localhost:$GOPHER_LOCALPORT/json 2>/dev/null
}

sendChannelMessage(){
	local CHATUSER CHANNEL
	GOPHER_MESSAGE_FORMAT=${GOPHER_MESSAGE_FORMAT:-variable}
	CHANNEL=$1
	shift
	MESSAGE="$*"
	MESSAGE=$(encode "$MESSAGE")

JSON=$(cat <<EOF
{
	"Command": "SendChannelMessage",
	"CmdArgs": {
		"Channel": "$CHANNEL",
		"Format": "$GOPHER_MESSAGE_FORMAT",
		"Message": "$MESSAGE"
	}
}
EOF
)
	echo "$JSON" | curl -X POST -d @- http://localhost:$GOPHER_LOCALPORT/json 2>/dev/null
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
		sendUserChannelMessage "$CHATUSER" "$CHANNEL" "$*"
	else
		sendUserMessage "$CHATUSER" "$*"
	fi
}
