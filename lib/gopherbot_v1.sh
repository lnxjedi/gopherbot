#!/bin/bash
# shellLib.sh - bash plugins should source this with 'source $GOPHER_INSTALLDIR/util/shellLib.sh'
# Return values for robot method calls
GBRET_Ok=0
GBRET_UserNotFound=1
GBRET_ChannelNotFound=2
GBRET_AttributeNotFound=3
GBRET_FailedMessageSend=4
GBRET_FailedChannelJoin=5
GBRET_DatumNotFound=6
GBRET_DatumLockExpired=7
GBRET_DataFormatError=8
GBRET_BrainFailed=9
GBRET_InvalidDatumKey=10
GBRET_InvalidDblPtr=11
GBRET_InvalidCfgStruct=12
GBRET_NoConfigFound=13
GBRET_RetryPrompt=14
GBRET_ReplyNotMatched=15
GBRET_UseDefaultValue=16
GBRET_TimeoutExpired=17
GBRET_Interrupted=18
GBRET_MatcherNotFound=19
GBRET_NoUserEmail=20
GBRET_NoBotEmail=21
GBRET_MailError=22
GBRET_TaskNotFound=23
GBRET_MissingArguments=24
GBRET_InvalidStage=25

# Plugin return values / exit codes
PLUGRET_Normal=0
PLUGRET_Fail=1
PLUGRET_MechanismFail=2
PLUGRET_ConfigurationError=3
PLUGRET_NotFound=6
PLUGRET_Success=7

base64_encode(){
	local MESSAGE
	MESSAGE=$(echo -n "$@" | base64)
	MESSAGE=$(echo -n "$MESSAGE")
	echo -n "$MESSAGE"
}

# Create the full JSON string and post it
gbPostJSON(){
	local GB_FUNCNAME=$1
	local GB_FUNCARGS="$2"
	local FORMAT=${3:-$GB_FORMAT}
	local JSON JSONRET
	#local GB_DEBUG="true"
	JSON=$(cat <<EOF
{
	"FuncName": "$GB_FUNCNAME",
	"User": "$GOPHER_USER",
	"Channel": "$GOPHER_CHANNEL",
	"Format": "$FORMAT",
	"Protocol": "$GOPHER_PROTOCOL",
	"CallerID": "$GOPHER_CALLER_ID",
	"FuncArgs": $GB_FUNCARGS
}
EOF
)
	if [ "$GB_DEBUG" = "true" ]
	then
		echo "Sending:" >&2
		echo "$JSON" >&2
	fi
	JSONRET=$(echo "$JSON" | curl -f -X POST -d @- $GOPHER_HTTP_POST/json 2>/dev/null)
	if [ "$GB_DEBUG" = "true" ]
	then
		echo "Got back:" >&2
		echo "$JSONRET" >&2
	fi
	echo "$JSONRET"
}

gbBotRet() {
	local JSON="$1"
	local RETVAL
	RETVAL=$(echo "$JSON" | jq .RetVal)
	return $RETVAL
}

gbExtract() {
	local JSON="$1"
	local ITEM="$2"
	echo "$JSON" | jq -r .$ITEM
}

CheckAdmin(){
	local GB_FUNCARGS="{}"
	local GB_FUNCNAME="CheckAdmin"
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	local RETVAL=$(echo "$GB_RET" | jq .Boolean)
	echo "$RETVAL"
	if [ "$RETVAL" -eq "true" ]
	then
		return 0
	else
		return 1
	fi
}

Remember(){
	if [ -z "$1" -o -z "$2" ]
	then
		return 1
	fi
	local GB_FUNCNAME="Remember"
	local R_KEY=$(base64_encode "$1")
	local R_MEMORY=$(base64_encode "$2")
	local GB_FUNCARGS=$(cat <<EOF
{
	"Key": "$R_KEY",
	"Value": "$R_MEMORY",
	"Base64" : true
}
EOF
)
	gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS"
	return 0
}

RememberContext(){
	if [ -z "$1" -o -z "$2" ]
	then
		return 1
	fi
	Remember "context:$1" "$2"
	return 0
}

Pause(){
	sleep $1
}

Recall(){
	if [ -z "$1" ]
	then
		return 1
	fi
	local R_KEY=$(base64_encode "$1")
	local GB_FUNCNAME="Recall"
	local GB_FUNCARGS=$(cat <<EOF
{
	"Key": "$R_KEY",
	"Base64": true
}
EOF
)
	local GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	local RETVAL=$(echo "$GB_RET" | jq -r .StrVal)
	echo -n "$RETVAL"
}

SetParameter() {
	local NAME=$(base64_encode "$1")
	local VALUE=$(base64_encode "$2")
	local GB_FUNCARGS=$(cat <<EOF
{
	"Name": "$NAME",
	"Value": "$VALUE",
	"Base64": true
}
EOF
)
	local GB_FUNCNAME="SetParameter"
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS" $FORMAT)
	local RETVAL=$(echo "$GB_RET" | jq .Boolean)
	if [ "$RETVAL" = "true" ]
	then
		return 0
	else
		return 1
	fi
}

ExtendNamespace() {
	local NS="$1"
	local HIST="$2"
	local GB_FUNCARGS=$(cat <<EOF
{
	"Extend": "$NS",
	"Histories": $2
}
EOF
)
	local GB_FUNCNAME="ExtendNamespace"
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS" $FORMAT)
	local RETVAL=$(echo "$GB_RET" | jq .Boolean)
	if [ "$RETVAL" = "true" ]
	then
		return 0
	else
		return 1
	fi
}

SetWorkingDirectory() {
	local WDPATH="$1"
	local GB_FUNCARGS=$(cat <<EOF
{
	"Path": "$WDPATH"
}
EOF
)
	local GB_FUNCNAME="SetWorkingDirectory"
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS" $FORMAT)
	local RETVAL=$(echo "$GB_RET" | jq .Boolean)
	if [ "$RETVAL" = "true" ]
	then
		return 0
	else
		return 1
	fi
}

_pipeTask(){
	local JSTR
	local FNAME="$1"
	local TNAME="$2"
	shift 2
	for ARG in "$@"
	do
		JSTR="$JSTR \"$ARG\""
	done
	if [ -n "$JSTR" ]
	then
		JSTR=$(echo ${JSTR//\" \"/\", \"})
	fi
	local GB_FUNCARGS=$(cat <<EOF
{
	"Name": "$TNAME",
	"CmdArgs": [ $JSTR ]
}
EOF
)
	GB_RET=$(gbPostJSON $FNAME "$GB_FUNCARGS" $FORMAT)
	gbBotRet "$GB_RET"
}

AddJob(){
	_pipeTask "AddJob" "$@"
}

AddTask(){
	_pipeTask "AddTask" "$@"
}

FinalTask(){
	_pipeTask "FinalTask" "$@"
}

FailTask(){
	_pipeTask "FailTask" "$@"
}

SpawnJob(){
	_pipeTask "SpawnJob" "$@"
}

_cmdTask(){
	local JSTR
	local FNAME="$1"
	local TNAME="$2"
	local PCMD="$3"
	local GB_FUNCARGS=$(cat <<EOF
{
	"Plugin": "$TNAME",
	"Command": "$PCMD"
}
EOF
)
	GB_RET=$(gbPostJSON $FNAME "$GB_FUNCARGS" $FORMAT)
	gbBotRet "$GB_RET"
}

AddCommand(){
	_cmdTask "AddCommand" "$@"
}

FailCommand(){
	_cmdTask "FailCommand" "$@"
}

FinalCommand(){
	_cmdTask "FinalCommand" "$@"
}

Exclusive(){
	local QUEUE_TASK="false"
	local TAG="$1"
	if [ -n "$2" ]
	then
		QUEUE_TASK=$2
	fi
	local GB_FUNCARGS=$(cat <<EOF
{
	"Tag": "$TAG",
	"QueueTask": $QUEUE_TASK
}
EOF
)
	local GB_FUNCNAME="Exclusive"
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	local RETVAL=$(echo "$GB_RET" | jq .Boolean)
	if [ "$RETVAL" = "true" ]
	then
		return 0
	else
		return 1
	fi
}

Elevate(){
	IMMEDIATE="false"
	if [ -n "$1" ]
	then
		IMMEDIATE = $1
	fi
	local GB_FUNCARGS=$(cat <<EOF
{
	"Immediate": "$IMMEDIATE"
}
EOF
)
	local GB_FUNCNAME="Elevate"
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	local RETVAL=$(echo "$GB_RET" | jq .Boolean)
	if [ "$RETVAL" = "true" ]
	then
		return 0
	else
		return 1
	fi
}

GetBotAttribute(){
	local GB_FUNCARGS GB_RET
	local GB_FUNCNAME="GetBotAttribute"
	local ATTR="$1"
	GB_FUNCARGS=$(cat <<EOF
{
	"Attribute": "$ATTR"
}
EOF
)
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbExtract "$GB_RET" Attribute
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
	gbExtract "$GB_RET" Attribute
	gbBotRet "$GB_RET"
}

GetUserAttribute(){
	local GB_FUNCARGS GB_RET
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
	gbExtract "$GB_RET" Attribute
	gbBotRet "$GB_RET"
}

Log(){
	local GB_FUNCARGS GB_RET
	local GB_FUNCNAME="Log"
	local GLM_LEVEL="$1"
	local GLM_MESSAGE=$(base64_encode "$2")
	GB_FUNCARGS=$(cat <<EOF
{
	"Level": "$GLM_LEVEL",
	"Message": "$GLM_MESSAGE",
	"Base64" : true
}
EOF
)
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS")
	gbBotRet "$GB_RET"
}

PromptUserChannelForReply(){
	local FORMAT
	if [[ $1 = -? ]]; then FORMAT=$(getFormat $1); shift; fi
	local GB_FUNCARGS GB_RET
	local GB_FUNCNAME="PromptUserChannelForReply"
	local REGEX="$1"
	local PUSER="$2"
	local PCHANNEL="$3"
	local PROMPT=$(base64_encode "$4")
	GB_FUNCARGS=$(cat <<EOF
{
	"RegexID": "$REGEX",
	"User": "$PUSER",
	"Channel": "$PCHANNEL",
	"Prompt": "$PROMPT",
	"Base64" : true
}
EOF
)
	local RETVAL
	for TRY in 0 1 2
	do
		GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS" $FORMAT)
		gbBotRet "$GB_RET"
		RETVAL=$?
		if [ $RETVAL -eq $GBRET_RetryPrompt ]
		then
			continue
		fi
		gbExtract "$GB_RET" Reply
		return $RETVAL
	done
	gbBotRet "$GB_RET"
	RETVAL=$?
	if [ $RETVAL -eq $GBRET_RetryPrompt ]
	then
		return $GBRET_Interrupted
	else
		return $RETVAL
	fi
}

PromptForReply(){
	local FORMAT
	if [[ $1 = -? ]]; then FORMAT=$1; shift; fi
	local REGEX=$1
	shift
	PromptUserChannelForReply $FORMAT "$REGEX" "$GOPHER_USER" "$GOPHER_CHANNEL" "$*"
}

PromptUserForReply(){
	local FORMAT
	if [[ $1 = -? ]]; then FORMAT=$1; shift; fi
	local REGEX=$1
	local PUSER=$2
	shift 2
	PromptUserChannelForReply "$REGEX" "$PUSER" "" "$*"
}

MessageFormat(){
	if [ -n "$1" ]
	then
		export GB_FORMAT="$1"
	fi
}

getFormat(){
	case "$1" in
	"-f")
		echo "Fixed"
		;;
	"-r")
		echo "Raw"
		;;
	"-v")
		echo "Variable"
		;;
	esac
}

SendUserMessage(){
	local FORMAT
	if [[ $1 = -? ]]; then FORMAT=$(getFormat $1); shift; fi
	local GB_FUNCARGS GB_RET
	local GB_FUNCNAME="SendUserMessage"
	local SUM_USER=$1
	shift
	MESSAGE="$*"
	MESSAGE=$(base64_encode "$MESSAGE")

	GB_FUNCARGS=$(cat <<EOF
{
	"User": "$SUM_USER",
	"Message": "$MESSAGE",
	"Base64" : true
}
EOF
)
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS" $FORMAT)
	gbBotRet "$GB_RET"
}

SendUserChannelMessage(){
	local FORMAT
	if [[ $1 = -? ]]; then FORMAT=$(getFormat $1); shift; fi
	local GB_FUNCARGS GB_RET
	local GB_FUNCNAME="SendUserChannelMessage"
	local SUCM_USER=$1
	local SUCM_CHANNEL=$2
	shift 2
	MESSAGE="$*"
	MESSAGE=$(base64_encode "$MESSAGE")

	GB_FUNCARGS=$(cat <<EOF
{
	"User": "$SUCM_USER",
	"Channel": "$SUCM_CHANNEL",
	"Message": "$MESSAGE",
	"Base64" : true
}
EOF
)
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS" $FORMAT)
	gbBotRet "$GB_RET"
}

SendChannelMessage(){
	local FORMAT
	if [[ $1 = -? ]]; then FORMAT=$(getFormat $1); shift; fi
	local GB_FUNCARGS GB_RET
	local GB_FUNCNAME="SendChannelMessage"
	local SCM_CHANNEL=$1
	shift
	MESSAGE="$*"
	MESSAGE=$(base64_encode "$MESSAGE")

	GB_FUNCARGS=$(cat <<EOF
{
	"Channel": "$SCM_CHANNEL",
	"Message": "$MESSAGE",
	"Base64" : true
}
EOF
)
	GB_RET=$(gbPostJSON $GB_FUNCNAME "$GB_FUNCARGS" $FORMAT)
	gbBotRet "$GB_RET"
}

# Convenience functions so that copies of this logic don't wind up in a bunch of plugins
Say(){
	local FARG
	[[ $1 == -? ]] && { FARG=$1; shift; }
	if [ -n "$GOPHER_CHANNEL" ]
	then
		SendChannelMessage $FARG "$GOPHER_CHANNEL" "$*"
	else
		SendUserMessage $FARG "$GOPHER_USER" "$*"
	fi
}

Reply(){
	local FARG
	[[ $1 == -? ]] && { FARG=$1; shift; }
	if [ -n "$GOPHER_CHANNEL" ]
	then
		SendUserChannelMessage $FARG "$GOPHER_USER" "$GOPHER_CHANNEL" "$*"
	else
		SendUserMessage $FARG "$GOPHER_USER" "$*"
	fi
}
