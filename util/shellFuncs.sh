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

GetAttribute(){
	local JSON
	local ATTR="$1"
	JSON=$(cat <<EOF
{
	"Command": "GetAttribute",
	"CmdArgs": {
		"Attribute": "$ATTR"
	}
}
EOF
)
	echo "$JSON" | curl -X POST -d @- http://localhost:$GOPHER_LOCALPORT/json 2>/dev/null
}

GetUserAttribute(){
	local JSON
	local GETUSER="$1"
	local ATTR="$2"
	JSON=$(cat <<EOF
{
	"Command": "GetAttribute",
	"CmdArgs": {
		"User": "$GETUSER",
		"Attribute": "$ATTR"
	}
}
EOF
)
	echo "$JSON" | curl -X POST -d @- http://localhost:$GOPHER_LOCALPORT/json 2>/dev/null
}

SendUserMessage(){
	local JSON
	[ "$1" = "-f" ] && { GOPHER_MESSAGE_FORMAT=fixed; shift; }
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

SendUserChannelMessage(){
	local JSON
	[ "$1" = "-f" ] && { GOPHER_MESSAGE_FORMAT=fixed; shift; }
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

SendChannelMessage(){
	local JSON
	[ "$1" = "-f" ] && { GOPHER_MESSAGE_FORMAT=fixed; shift; }
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
Say(){
	[ "$1" = "-f" ] && { GOPHER_MESSAGE_FORMAT=fixed; shift; }
	if [ -n "$CHANNEL" ]
	then
		SendChannelMessage "$CHANNEL" "$*"
	else
		SendUserMessage "$CHATUSER" "$*"
	fi
}

Reply(){
	[ "$1" = "-f" ] && { GOPHER_MESSAGE_FORMAT=fixed; shift; }
	if [ -n "$CHANNEL" ]
	then
		SendUserChannelMessage "$CHATUSER" "$CHANNEL" "$*"
	else
		SendUserMessage "$CHATUSER" "$*"
	fi
}
