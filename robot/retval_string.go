// Code generated by "stringer -type=RetVal botdefs.go"; DO NOT EDIT.

package robot

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Ok-0]
	_ = x[UserNotFound-1]
	_ = x[ChannelNotFound-2]
	_ = x[AttributeNotFound-3]
	_ = x[FailedMessageSend-4]
	_ = x[FailedChannelJoin-5]
	_ = x[DatumNotFound-6]
	_ = x[DatumLockExpired-7]
	_ = x[DataFormatError-8]
	_ = x[BrainFailed-9]
	_ = x[InvalidDatumKey-10]
	_ = x[InvalidDblPtr-11]
	_ = x[InvalidCfgStruct-12]
	_ = x[NoConfigFound-13]
	_ = x[RetryPrompt-14]
	_ = x[ReplyNotMatched-15]
	_ = x[UseDefaultValue-16]
	_ = x[TimeoutExpired-17]
	_ = x[Interrupted-18]
	_ = x[MatcherNotFound-19]
	_ = x[NoUserEmail-20]
	_ = x[NoBotEmail-21]
	_ = x[MailError-22]
	_ = x[TaskNotFound-23]
	_ = x[MissingArguments-24]
	_ = x[InvalidStage-25]
	_ = x[InvalidTaskType-26]
	_ = x[CommandNotMatched-27]
	_ = x[TaskDisabled-28]
}

const _RetVal_name = "OkUserNotFoundChannelNotFoundAttributeNotFoundFailedMessageSendFailedChannelJoinDatumNotFoundDatumLockExpiredDataFormatErrorBrainFailedInvalidDatumKeyInvalidDblPtrInvalidCfgStructNoConfigFoundRetryPromptReplyNotMatchedUseDefaultValueTimeoutExpiredInterruptedMatcherNotFoundNoUserEmailNoBotEmailMailErrorTaskNotFoundMissingArgumentsInvalidStageInvalidTaskTypeCommandNotMatchedTaskDisabled"

var _RetVal_index = [...]uint16{0, 2, 14, 29, 46, 63, 80, 93, 109, 124, 135, 150, 163, 179, 192, 203, 218, 233, 247, 258, 273, 284, 294, 303, 315, 331, 343, 358, 375, 387}

func (i RetVal) String() string {
	if i < 0 || i >= RetVal(len(_RetVal_index)-1) {
		return "RetVal(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _RetVal_name[_RetVal_index[i]:_RetVal_index[i+1]]
}