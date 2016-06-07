#!/bin/bash
# shellLib.sh - bash plugins should source this with 'source $GOPHER_INSTALLDIR/util/shellLib.sh'

GB_CHANNEL=$1
GB_USER=$2
GB_PLUGID=$3
export GB_CHANNEL GB_USER GB_PLUGID
shift 3
# Now $1 is the command

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
	local JSON JSONRET
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
	JSONRET=$(echo "$JSON" | curl -f -X POST -d @- $GOPHER_HTTP_POST/json 2>/dev/null)
	GB_FORMAT="variable"
}

gbBotRet() {
	local JSON="$1"
	local RETVAL
	RETVAL=$(echo "$JSON" | jq .BotRetVal)
	return $RETVAL
}

gbDecode() {
	local JSON="$1"
	local ITEM="$2"
	local B64DATA=$(echo "$JSON" | jq -r .$ITEM)
	B64DATA=${B64DATA#base64:}
	echo "$B64DATA" | base64 -d
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
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbDecode "$GB_RET" Attribute
	gbBotRet "$GB_RET"
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
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbDecode "$GB_RET" Attribute
	gbBotRet "$GB_RET"
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
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbDecode "$GB_RET" Attribute
	gbBotRet "$GB_RET"
}

WaitForReply(){
	local GB_FUNCARGS
	local GB_FUNCNAME="WaitForReply"
	local REGEX="$1"
	local TIMEOUT="${2:-14}"
	GB_FUNCARGS=$(cat <<EOF
{
	"RegExId": "$REGEX",
	"Timeout": $TIMEOUT
}
EOF
)
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbDecode "$GB_RET" Reply
	gbBotRet "$GB_RET"
}

SendUserMessage(){
	[ "$1" = "-f" ] && { GB_FORMAT=fixed; shift; }
	local GB_FUNCARGS GB_RET
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
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbBotRet "$GB_RET"
}

SendUserChannelMessage(){
	[ "$1" = "-f" ] && { GB_FORMAT=fixed; shift; }
	local GB_FUNCARGS GB_RET
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
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbBotRet "$GB_RET"
}

SendChannelMessage(){
	[ "$1" = "-f" ] && { GB_FORMAT=fixed; shift; }
	local GB_FUNCARGS GB_RET
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
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbBotRet "$GB_RET"
}

# Convenience functions so that copies of this logic don't wind up in a bunch of plugins
Say(){
	[ "$1" = "-f" ] && { GB_FORMAT=fixed; shift; }
	if [ -n "$GB_CHANNEL" ]
	then
		SendChannelMessage "$GB_CHANNEL" "$*"
	else
		SendUserMessage "$GB_USER" "$*"
	fi
}

Reply(){
	[ "$1" = "-f" ] && { GB_FORMAT=fixed; shift; }
	if [ -n "$GB_CHANNEL" ]
	then
		SendUserChannelMessage "$GB_USER" "$GB_CHANNEL" "$*"
	else
		SendUserMessage "$GB_USER" "$*"
	fi
}
