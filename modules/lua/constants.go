package lua

import (
	"github.com/lnxjedi/gopherbot/robot"
	glua "github.com/yuin/gopher-lua"
)

// registerConstants registers a full set of Robot constants
// into the global Lua environment.
func registerConstants(L *glua.LState) {
	//----------------------------------------------------------------------
	// 1. RetVal (bot method returns) => "retXYZ"
	//----------------------------------------------------------------------
	L.SetGlobal("retOk", glua.LNumber(robot.Ok))
	L.SetGlobal("retUserNotFound", glua.LNumber(robot.UserNotFound))
	L.SetGlobal("retChannelNotFound", glua.LNumber(robot.ChannelNotFound))
	L.SetGlobal("retAttributeNotFound", glua.LNumber(robot.AttributeNotFound))
	L.SetGlobal("retFailedMessageSend", glua.LNumber(robot.FailedMessageSend))
	L.SetGlobal("retFailedChannelJoin", glua.LNumber(robot.FailedChannelJoin))

	// Brain maladies
	L.SetGlobal("retDatumNotFound", glua.LNumber(robot.DatumNotFound))
	L.SetGlobal("retDatumLockExpired", glua.LNumber(robot.DatumLockExpired))
	L.SetGlobal("retDataFormatError", glua.LNumber(robot.DataFormatError))
	L.SetGlobal("retBrainFailed", glua.LNumber(robot.BrainFailed))
	L.SetGlobal("retInvalidDatumKey", glua.LNumber(robot.InvalidDatumKey))

	// GetTaskConfig
	L.SetGlobal("retInvalidConfigPointer", glua.LNumber(robot.InvalidConfigPointer))
	L.SetGlobal("retConfigUnmarshalError", glua.LNumber(robot.ConfigUnmarshalError))
	L.SetGlobal("retNoConfigFound", glua.LNumber(robot.NoConfigFound))

	// PromptForReply
	L.SetGlobal("retRetryPrompt", glua.LNumber(robot.RetryPrompt))
	L.SetGlobal("retReplyNotMatched", glua.LNumber(robot.ReplyNotMatched))
	L.SetGlobal("retUseDefaultValue", glua.LNumber(robot.UseDefaultValue))
	L.SetGlobal("retTimeoutExpired", glua.LNumber(robot.TimeoutExpired))
	L.SetGlobal("retInterrupted", glua.LNumber(robot.Interrupted))
	L.SetGlobal("retMatcherNotFound", glua.LNumber(robot.MatcherNotFound))

	// Email
	L.SetGlobal("retNoUserEmail", glua.LNumber(robot.NoUserEmail))
	L.SetGlobal("retNoBotEmail", glua.LNumber(robot.NoBotEmail))
	L.SetGlobal("retMailError", glua.LNumber(robot.MailError))

	// Pipeline errors
	L.SetGlobal("retTaskNotFound", glua.LNumber(robot.TaskNotFound))
	L.SetGlobal("retMissingArguments", glua.LNumber(robot.MissingArguments))
	L.SetGlobal("retInvalidStage", glua.LNumber(robot.InvalidStage))
	L.SetGlobal("retInvalidTaskType", glua.LNumber(robot.InvalidTaskType))
	L.SetGlobal("retCommandNotMatched", glua.LNumber(robot.CommandNotMatched))
	L.SetGlobal("retTaskDisabled", glua.LNumber(robot.TaskDisabled))
	L.SetGlobal("retPrivilegeViolation", glua.LNumber(robot.PrivilegeViolation))

	//----------------------------------------------------------------------
	// 2. TaskRetVal (script return values) => "taskXYZ"
	//----------------------------------------------------------------------
	L.SetGlobal("taskNormal", glua.LNumber(robot.Normal)) // 0
	L.SetGlobal("taskFail", glua.LNumber(robot.Fail))     // 1
	L.SetGlobal("taskMechanismFail", glua.LNumber(robot.MechanismFail))
	L.SetGlobal("taskConfigurationError", glua.LNumber(robot.ConfigurationError))
	L.SetGlobal("taskPipelineAborted", glua.LNumber(robot.PipelineAborted))
	L.SetGlobal("taskRobotStopping", glua.LNumber(robot.RobotStopping))
	L.SetGlobal("taskNotFound", glua.LNumber(robot.NotFound)) // NOTE: 'NotFound' also used above as a pipeline error
	L.SetGlobal("taskSuccess", glua.LNumber(robot.Success))   // 7

	//----------------------------------------------------------------------
	// 3. LogLevel => "logXYZ"
	//----------------------------------------------------------------------
	L.SetGlobal("logTrace", glua.LNumber(robot.Trace))
	L.SetGlobal("logDebug", glua.LNumber(robot.Debug))
	L.SetGlobal("logInfo", glua.LNumber(robot.Info))
	L.SetGlobal("logAudit", glua.LNumber(robot.Audit))
	L.SetGlobal("logWarn", glua.LNumber(robot.Warn))
	L.SetGlobal("logError", glua.LNumber(robot.Error))
	L.SetGlobal("logFatal", glua.LNumber(robot.Fatal))

	//----------------------------------------------------------------------
	// 4. MessageFormat => "fmtXYZ"
	//----------------------------------------------------------------------
	L.SetGlobal("fmtRaw", glua.LNumber(robot.Raw))
	L.SetGlobal("fmtFixed", glua.LNumber(robot.Fixed))
	L.SetGlobal("fmtVariable", glua.LNumber(robot.Variable))

	//----------------------------------------------------------------------
	// 5. Protocol => "protoXYZ"
	//----------------------------------------------------------------------
	L.SetGlobal("protoSlack", glua.LNumber(robot.Slack))
	L.SetGlobal("protoRocket", glua.LNumber(robot.Rocket))
	L.SetGlobal("protoTerminal", glua.LNumber(robot.Terminal))
	L.SetGlobal("protoTest", glua.LNumber(robot.Test))
	L.SetGlobal("protoNull", glua.LNumber(robot.Null))
}
