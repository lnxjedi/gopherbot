# gopherbot_v1.gsh
#
# Lightweight compatibility shim for the built-in Gopherbot shell runtime.
# Robot API methods are provided as built-in commands by the interpreter; this
# file provides the familiar return-code constants plus MessageFormat state.

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
GBRET_InvalidConfigPointer=11
GBRET_ConfigUnmarshalError=12
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
GBRET_InvalidTaskType=26
GBRET_CommandNotMatched=27
GBRET_TaskDisabled=28
GBRET_PrivilegeViolation=29
GBRET_Failed=63

PLUGRET_Normal=0
PLUGRET_Fail=1
PLUGRET_MechanismFail=2
PLUGRET_ConfigurationError=3
PLUGRET_NotFound=6
PLUGRET_Success=7

MessageFormat() {
	if [ -n "$1" ]
	then
		export GBOT_MESSAGE_FORMAT="$1"
	fi
}
