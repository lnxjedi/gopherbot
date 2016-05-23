gb_json_encode(){
	local MESSAGE
	MESSAGE=$(echo "$@" | base64)
	MESSAGE=$(echo "base64:$MESSAGE")
	echo "$MESSAGE"
}

# Create the full JSON string and post it
gbPostJSON(){
	local GB_FUNCNAME=$1
	local GB_FUNCARGS="$2"
	local JSON
	GB_FORMAT=${GB_FORMAT:-variable}
	JSON=$(cat <<EOF
{
	"FuncName": "$GB_FUNCNAME",
	"User": "$GB_USER",
	"Channel": "$GB_CHANNEL",
	"Format": "$GB_FORMAT",
	"PluginID": "$GB_PLUGID",
	"FuncArgs": $GB_FUNCARGS
}
EOF
)
	echo "$JSON" | curl -f -X POST -d @- $GOPHER_HTTP_POST/json 2>/dev/null
}

GetBotAttribute(){
	local GB_FUNCARGS
	local GB_FUNCNAME="GetBotAttribute"
	local ATTR="$1"
	GB_FUNCARGS=$(cat <<EOF
{
	"Attribute": "$ATTR"
}
EOF
)
	gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS"
}

GetSenderAttribute(){
	local GB_FUNCARGS
	local GB_FUNCNAME="GetSenderAttribute"
	local ATTR="$1"
	GB_FUNCARGS=$(cat <<EOF
{
	"Attribute": "$ATTR"
}
EOF
)
	gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS"
}

GetUserAttribute(){
	local GB_FUNCARGS
	local GB_FUNCNAME="GetUserAttribute"
	local GUA_USER="$1"
	local ATTR="$2"
	GB_FUNCARGS=$(cat <<EOF
{
	"User": "$GUA_USER",
	"Attribute": "$ATTR"
}
EOF
)
	gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS"
}

WaitForReply(){
	local GB_FUNCARGS
	local GB_FUNCNAME="WaitForReply"
	local REGEX="$1"
	local TIMEOUT="${2:-14}"
	local NEEDCMD="${3:-false}"
	GB_FUNCARGS=$(cat <<EOF
{
	"RegExId": "$REGEX",
	"Timeout": $TIMEOUT,
	"NeedCommand": $NEEDCMD
}
EOF
)
	gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS"
}

SendUserMessage(){
	[ "$1" = "-f" ] && { GB_FORMAT=fixed; shift; }
	local GB_FUNCARGS GB_RETVAL
	local GB_FUNCNAME="SendUserMessage"
	local SUM_USER=$1
	shift
	MESSAGE="$*"
	MESSAGE=$(gb_json_encode "$MESSAGE")

	GB_FUNCARGS=$(cat <<EOF
{
	"User": "$SUM_USER",
	"Message": "$MESSAGE"
}
EOF
)
	gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS"
	GB_RETVAL=$?; GB_FORMAT=variable; return $GB_RETVAL
}

SendUserChannelMessage(){
	[ "$1" = "-f" ] && { GB_FORMAT=fixed; shift; }
	local GB_FUNCARGS GB_RETVAL
	local GB_FUNCNAME="SendUserChannelMessage"
	local SUCM_USER=$1
	local SUCM_CHANNEL=$2
	shift 2
	MESSAGE="$*"
	MESSAGE=$(gb_json_encode "$MESSAGE")

	GB_FUNCARGS=$(cat <<EOF
{
	"User": "$SUCM_USER",
	"Channel": "$SUCM_CHANNEL",
	"Message": "$MESSAGE"
}
EOF
)
	gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS"
	GB_RETVAL=$?; GB_FORMAT=variable; return $GB_RETVAL
}

SendChannelMessage(){
	[ "$1" = "-f" ] && { GB_FORMAT=fixed; shift; }
	local GB_FUNCARGS GB_RETVAL
	local GB_FUNCNAME="SendChannelMessage"
	local SCM_CHANNEL=$1
	shift
	MESSAGE="$*"
	MESSAGE=$(gb_json_encode "$MESSAGE")

	GB_FUNCARGS=$(cat <<EOF
{
	"Channel": "$SCM_CHANNEL",
	"Message": "$MESSAGE"
}
EOF
)
	gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS"
	GB_RETVAL=$?; GB_FORMAT=variable; return $GB_RETVAL
}

# Convenience functions so that copies of this logic don't wind up in a bunch of plugins
Say(){
	local GB_RETVAL
	[ "$1" = "-f" ] && { GB_FORMAT=fixed; shift; }
	if [ -n "$GB_CHANNEL" ]
	then
		SendChannelMessage "$GB_CHANNEL" "$*"
	else
		SendUserMessage "$GB_USER" "$*"
	fi
	GB_RETVAL=$?; GB_FORMAT=variable; return $GB_RETVAL
}

Reply(){
	local GB_RETVAL
	[ "$1" = "-f" ] && { GB_FORMAT=fixed; shift; }
	if [ -n "$GB_CHANNEL" ]
	then
		SendUserChannelMessage "$GB_USER" "$GB_CHANNEL" "$*"
	else
		SendUserMessage "$GB_USER" "$*"
	fi
	GB_RETVAL=$?; GB_FORMAT=variable; return $GB_RETVAL
}
