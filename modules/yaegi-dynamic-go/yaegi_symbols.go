// Code generated by 'yaegi extract github.com/lnxjedi/gopherbot/robot'. DO NOT EDIT.

package yaegidynamicgo

import (
	"bytes"
	"github.com/lnxjedi/gopherbot/robot"
	"go/constant"
	"go/token"
	"io"
	"reflect"
)

var Symbols = map[string]map[string]reflect.Value{}

func init() {
	Symbols["github.com/lnxjedi/gopherbot/robot/robot"] = map[string]reflect.Value{
			// function, constant and variable definitions
			"AttributeNotFound": reflect.ValueOf(robot.AttributeNotFound),
					"Audit": reflect.ValueOf(robot.Audit),
					"BrainFailed": reflect.ValueOf(robot.BrainFailed),
					"ChannelNotFound": reflect.ValueOf(robot.ChannelNotFound),
					"CommandNotMatched": reflect.ValueOf(robot.CommandNotMatched),
					"ConfigurationError": reflect.ValueOf(robot.ConfigurationError),
					"DataFormatError": reflect.ValueOf(robot.DataFormatError),
					"DatumLockExpired": reflect.ValueOf(robot.DatumLockExpired),
					"DatumNotFound": reflect.ValueOf(robot.DatumNotFound),
					"Debug": reflect.ValueOf(robot.Debug),
					"Error": reflect.ValueOf(robot.Error),
					"Fail": reflect.ValueOf(robot.Fail),
					"FailedChannelJoin": reflect.ValueOf(robot.FailedChannelJoin),
					"FailedMessageSend": reflect.ValueOf(robot.FailedMessageSend),
					"Fatal": reflect.ValueOf(robot.Fatal),
					"Fixed": reflect.ValueOf(robot.Fixed),
					"GetRegistrations": reflect.ValueOf(robot.GetRegistrations),
					"Info": reflect.ValueOf(robot.Info),
					"Interrupted": reflect.ValueOf(robot.Interrupted),
					"InvalidConfigPointer": reflect.ValueOf(robot.InvalidConfigPointer),
					"InvalidDatumKey": reflect.ValueOf(robot.InvalidDatumKey),
					"ConfigUnmarshalError": reflect.ValueOf(robot.ConfigUnmarshalError),
					"InvalidStage": reflect.ValueOf(robot.InvalidStage),
					"InvalidTaskType": reflect.ValueOf(robot.InvalidTaskType),
					"MailError": reflect.ValueOf(robot.MailError),
					"MatcherNotFound": reflect.ValueOf(robot.MatcherNotFound),
					"MechanismFail": reflect.ValueOf(robot.MechanismFail),
					"MissingArguments": reflect.ValueOf(robot.MissingArguments),
					"NoBotEmail": reflect.ValueOf(robot.NoBotEmail),
					"NoConfigFound": reflect.ValueOf(robot.NoConfigFound),
					"NoUserEmail": reflect.ValueOf(robot.NoUserEmail),
					"Normal": reflect.ValueOf(robot.Normal),
					"NotFound": reflect.ValueOf(robot.NotFound),
					"Null": reflect.ValueOf(robot.Null),
					"Ok": reflect.ValueOf(robot.Ok),
					"PipelineAborted": reflect.ValueOf(robot.PipelineAborted),
					"PrivilegeViolation": reflect.ValueOf(robot.PrivilegeViolation),
					"Raw": reflect.ValueOf(robot.Raw),
					"RegisterJob": reflect.ValueOf(robot.RegisterJob),
					"RegisterPlugin": reflect.ValueOf(robot.RegisterPlugin),
					"RegisterTask": reflect.ValueOf(robot.RegisterTask),
					"ReplyNotMatched": reflect.ValueOf(robot.ReplyNotMatched),
					"RetryPrompt": reflect.ValueOf(robot.RetryPrompt),
					"RobotStopping": reflect.ValueOf(robot.RobotStopping),
					"Rocket": reflect.ValueOf(robot.Rocket),
					"Slack": reflect.ValueOf(robot.Slack),
					"Success": reflect.ValueOf(constant.MakeFromLiteral("7", token.INT, 0)),
					"TaskDisabled": reflect.ValueOf(robot.TaskDisabled),
					"TaskNotFound": reflect.ValueOf(robot.TaskNotFound),
					"Terminal": reflect.ValueOf(robot.Terminal),
					"Test": reflect.ValueOf(robot.Test),
					"TimeoutExpired": reflect.ValueOf(robot.TimeoutExpired),
					"Trace": reflect.ValueOf(robot.Trace),
					"UseDefaultValue": reflect.ValueOf(robot.UseDefaultValue),
					"UserNotFound": reflect.ValueOf(robot.UserNotFound),
					"Variable": reflect.ValueOf(robot.Variable),
					"Warn": reflect.ValueOf(robot.Warn),

			// type definitions
			"AttrRet": reflect.ValueOf((*robot.AttrRet)(nil)),
			"Connector": reflect.ValueOf((*robot.Connector)(nil)),
			"ConnectorMessage": reflect.ValueOf((*robot.ConnectorMessage)(nil)),
			"Handler": reflect.ValueOf((*robot.Handler)(nil)),
			"HistoryLogger": reflect.ValueOf((*robot.HistoryLogger)(nil)),
			"HistoryProvider": reflect.ValueOf((*robot.HistoryProvider)(nil)),
			"JobHandler": reflect.ValueOf((*robot.JobHandler)(nil)),
			"LogLevel": reflect.ValueOf((*robot.LogLevel)(nil)),
			"Logger": reflect.ValueOf((*robot.Logger)(nil)),
			"Message": reflect.ValueOf((*robot.Message)(nil)),
			"MessageFormat": reflect.ValueOf((*robot.MessageFormat)(nil)),
			"PluginHandler": reflect.ValueOf((*robot.PluginHandler)(nil)),
			"Protocol": reflect.ValueOf((*robot.Protocol)(nil)),
			"Registrations": reflect.ValueOf((*robot.Registrations)(nil)),
			"RetVal": reflect.ValueOf((*robot.RetVal)(nil)),
			"Robot": reflect.ValueOf((*robot.Robot)(nil)),
			"SimpleBrain": reflect.ValueOf((*robot.SimpleBrain)(nil)),
			"TaskHandler": reflect.ValueOf((*robot.TaskHandler)(nil)),
			"TaskRegistration": reflect.ValueOf((*robot.TaskRegistration)(nil)),
			"TaskRetVal": reflect.ValueOf((*robot.TaskRetVal)(nil)),

			// interface wrapper definitions
			"_Connector": reflect.ValueOf((*_github_com_lnxjedi_gopherbot_robot_Connector)(nil)),
			"_Handler": reflect.ValueOf((*_github_com_lnxjedi_gopherbot_robot_Handler)(nil)),
			"_HistoryLogger": reflect.ValueOf((*_github_com_lnxjedi_gopherbot_robot_HistoryLogger)(nil)),
			"_HistoryProvider": reflect.ValueOf((*_github_com_lnxjedi_gopherbot_robot_HistoryProvider)(nil)),
			"_Logger": reflect.ValueOf((*_github_com_lnxjedi_gopherbot_robot_Logger)(nil)),
			"_Robot": reflect.ValueOf((*_github_com_lnxjedi_gopherbot_robot_Robot)(nil)),
			"_SimpleBrain": reflect.ValueOf((*_github_com_lnxjedi_gopherbot_robot_SimpleBrain)(nil)),

	}
}
// _github_com_lnxjedi_gopherbot_robot_Connector is an interface wrapper for Connector type
	type _github_com_lnxjedi_gopherbot_robot_Connector struct {
			IValue interface{}
			WDefaultHelp func() ( []string)
			WFormatHelp func(a0 string) ( string)
			WGetProtocolUserAttribute func(user string, attr string) (value string, ret robot.RetVal)
			WJoinChannel func(c string) ( robot.RetVal)
			WMessageHeard func(user string, channel string) ()
			WRun func(stopchannel <-chan struct{}) ()
			WSendProtocolChannelThreadMessage func(channelname string, threadid string, msg string, format robot.MessageFormat, msgObject *robot.ConnectorMessage) ( robot.RetVal)
			WSendProtocolUserChannelThreadMessage func(userid string, username string, channelname string, threadid string, msg string, format robot.MessageFormat, msgObject *robot.ConnectorMessage) ( robot.RetVal)
			WSendProtocolUserMessage func(user string, msg string, format robot.MessageFormat, msgObject *robot.ConnectorMessage) ( robot.RetVal)
			WSetUserMap func(a0 map[string]string) ()

	}
	func (W _github_com_lnxjedi_gopherbot_robot_Connector) DefaultHelp() ( []string) {return W.WDefaultHelp()
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Connector) FormatHelp(a0 string) ( string) {return W.WFormatHelp(a0)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Connector) GetProtocolUserAttribute(user string, attr string) (value string, ret robot.RetVal) {return W.WGetProtocolUserAttribute(user, attr)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Connector) JoinChannel(c string) ( robot.RetVal) {return W.WJoinChannel(c)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Connector) MessageHeard(user string, channel string) () { W.WMessageHeard(user, channel)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Connector) Run(stopchannel <-chan struct{}) () { W.WRun(stopchannel)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Connector) SendProtocolChannelThreadMessage(channelname string, threadid string, msg string, format robot.MessageFormat, msgObject *robot.ConnectorMessage) ( robot.RetVal) {return W.WSendProtocolChannelThreadMessage(channelname, threadid, msg, format, msgObject)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Connector) SendProtocolUserChannelThreadMessage(userid string, username string, channelname string, threadid string, msg string, format robot.MessageFormat, msgObject *robot.ConnectorMessage) ( robot.RetVal) {return W.WSendProtocolUserChannelThreadMessage(userid, username, channelname, threadid, msg, format, msgObject)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Connector) SendProtocolUserMessage(user string, msg string, format robot.MessageFormat, msgObject *robot.ConnectorMessage) ( robot.RetVal) {return W.WSendProtocolUserMessage(user, msg, format, msgObject)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Connector) SetUserMap(a0 map[string]string) () { W.WSetUserMap(a0)
			}

// _github_com_lnxjedi_gopherbot_robot_Handler is an interface wrapper for Handler type
	type _github_com_lnxjedi_gopherbot_robot_Handler struct {
			IValue interface{}
			WExtractID func(u string) ( string,  bool)
			WGetBrainConfig func(a0 interface{}) ( error)
			WGetConfigPath func() ( string)
			WGetDirectory func(path string) ( error)
			WGetEventStrings func() ( *[]string)
			WGetHistoryConfig func(a0 interface{}) ( error)
			WGetInstallPath func() ( string)
			WGetLogLevel func() ( robot.LogLevel)
			WGetProtocolConfig func(a0 interface{}) ( error)
			WIncomingMessage func(a0 *robot.ConnectorMessage) ()
			WLog func(l robot.LogLevel, m string, v ...interface{}) ()
			WSetBotID func(id string) ()
			WSetBotMention func(mention string) ()
			WSetTerminalWriter func(a0 io.Writer) ()

	}
	func (W _github_com_lnxjedi_gopherbot_robot_Handler) ExtractID(u string) ( string,  bool) {return W.WExtractID(u)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Handler) GetBrainConfig(a0 interface{}) ( error) {return W.WGetBrainConfig(a0)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Handler) GetConfigPath() ( string) {return W.WGetConfigPath()
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Handler) GetDirectory(path string) ( error) {return W.WGetDirectory(path)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Handler) GetEventStrings() ( *[]string) {return W.WGetEventStrings()
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Handler) GetHistoryConfig(a0 interface{}) ( error) {return W.WGetHistoryConfig(a0)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Handler) GetInstallPath() ( string) {return W.WGetInstallPath()
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Handler) GetLogLevel() ( robot.LogLevel) {return W.WGetLogLevel()
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Handler) GetProtocolConfig(a0 interface{}) ( error) {return W.WGetProtocolConfig(a0)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Handler) IncomingMessage(a0 *robot.ConnectorMessage) () { W.WIncomingMessage(a0)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Handler) Log(l robot.LogLevel, m string, v ...interface{}) () { W.WLog(l, m, v...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Handler) SetBotID(id string) () { W.WSetBotID(id)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Handler) SetBotMention(mention string) () { W.WSetBotMention(mention)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Handler) SetTerminalWriter(a0 io.Writer) () { W.WSetTerminalWriter(a0)
			}

// _github_com_lnxjedi_gopherbot_robot_HistoryLogger is an interface wrapper for HistoryLogger type
	type _github_com_lnxjedi_gopherbot_robot_HistoryLogger struct {
			IValue interface{}
			WClose func() ()
			WFinalize func() ()
			WLine func(line string) ()
			WLog func(line string) ()

	}
	func (W _github_com_lnxjedi_gopherbot_robot_HistoryLogger) Close() () { W.WClose()
			}
	func (W _github_com_lnxjedi_gopherbot_robot_HistoryLogger) Finalize() () { W.WFinalize()
			}
	func (W _github_com_lnxjedi_gopherbot_robot_HistoryLogger) Line(line string) () { W.WLine(line)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_HistoryLogger) Log(line string) () { W.WLog(line)
			}

// _github_com_lnxjedi_gopherbot_robot_HistoryProvider is an interface wrapper for HistoryProvider type
	type _github_com_lnxjedi_gopherbot_robot_HistoryProvider struct {
			IValue interface{}
			WGetLog func(tag string, index int) ( io.Reader,  error)
			WGetLogURL func(tag string, index int) (URL string, exists bool)
			WMakeLogURL func(tag string, index int) (URL string, exists bool)
			WNewLog func(tag string, index int, maxHistories int) ( robot.HistoryLogger,  error)

	}
	func (W _github_com_lnxjedi_gopherbot_robot_HistoryProvider) GetLog(tag string, index int) ( io.Reader,  error) {return W.WGetLog(tag, index)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_HistoryProvider) GetLogURL(tag string, index int) (URL string, exists bool) {return W.WGetLogURL(tag, index)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_HistoryProvider) MakeLogURL(tag string, index int) (URL string, exists bool) {return W.WMakeLogURL(tag, index)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_HistoryProvider) NewLog(tag string, index int, maxHistories int) ( robot.HistoryLogger,  error) {return W.WNewLog(tag, index, maxHistories)
			}

// _github_com_lnxjedi_gopherbot_robot_Logger is an interface wrapper for Logger type
	type _github_com_lnxjedi_gopherbot_robot_Logger struct {
			IValue interface{}
			WLog func(l robot.LogLevel, m string, v ...interface{}) ()

	}
	func (W _github_com_lnxjedi_gopherbot_robot_Logger) Log(l robot.LogLevel, m string, v ...interface{}) () { W.WLog(l, m, v...)
			}

// _github_com_lnxjedi_gopherbot_robot_Robot is an interface wrapper for Robot type
	type _github_com_lnxjedi_gopherbot_robot_Robot struct {
			IValue interface{}
			WAddCommand func(a0 string, a1 string) ( robot.RetVal)
			WAddJob func(a0 string, a1 ...string) ( robot.RetVal)
			WAddTask func(a0 string, a1 ...string) ( robot.RetVal)
			WCheckAdmin func() ( bool)
			WCheckinDatum func(key string, locktoken string) ()
			WCheckoutDatum func(key string, datum interface{}, rw bool) (locktoken string, exists bool, ret robot.RetVal)
			WDirect func() ( robot.Robot)
			WElevate func(a0 bool) ( bool)
			WEmail func(subject string, messageBody *bytes.Buffer, html ...bool) (ret robot.RetVal)
			WEmailAddress func(address string, subject string, messageBody *bytes.Buffer, html ...bool) (ret robot.RetVal)
			WEmailUser func(user string, subject string, messageBody *bytes.Buffer, html ...bool) (ret robot.RetVal)
			WExclusive func(tag string, queueTask bool) (success bool)
			WFailCommand func(a0 string, a1 string) ( robot.RetVal)
			WFailTask func(a0 string, a1 ...string) ( robot.RetVal)
			WFinalCommand func(a0 string, a1 string) ( robot.RetVal)
			WFinalTask func(a0 string, a1 ...string) ( robot.RetVal)
			WFixed func() ( robot.Robot)
			WGetBotAttribute func(a string) ( *robot.AttrRet)
			WGetMessage func() ( *robot.Message)
			WGetParameter func(name string) ( string)
			WGetSenderAttribute func(a string) ( *robot.AttrRet)
			WGetTaskConfig func(cfgptr interface{}) ( robot.RetVal)
			WGetUserAttribute func(u string, a string) ( *robot.AttrRet)
			WLog func(l robot.LogLevel, m string, v ...interface{}) ( bool)
			WMessageFormat func(f robot.MessageFormat) ( robot.Robot)
			WPause func(s float64) ()
			WPromptForReply func(regexID string, prompt string, v ...interface{}) ( string,  robot.RetVal)
			WPromptThreadForReply func(regexID string, prompt string, v ...interface{}) ( string,  robot.RetVal)
			WPromptUserChannelForReply func(regexID string, user string, channel string, prompt string, v ...interface{}) ( string,  robot.RetVal)
			WPromptUserChannelThreadForReply func(regexID string, user string, channel string, thread string, prompt string, v ...interface{}) ( string,  robot.RetVal)
			WPromptUserForReply func(regexID string, user string, prompt string, v ...interface{}) ( string,  robot.RetVal)
			WRandomInt func(n int) ( int)
			WRandomString func(s []string) ( string)
			WRecall func(key string, shared bool) ( string)
			WRemember func(key string, value string, shared bool) ()
			WRememberContext func(context string, value string) ()
			WRememberContextThread func(context string, value string) ()
			WRememberThread func(key string, value string, shared bool) ()
			WReply func(msg string, v ...interface{}) ( robot.RetVal)
			WReplyThread func(msg string, v ...interface{}) ( robot.RetVal)
			WSay func(msg string, v ...interface{}) ( robot.RetVal)
			WSayThread func(msg string, v ...interface{}) ( robot.RetVal)
			WSendChannelMessage func(ch string, msg string, v ...interface{}) ( robot.RetVal)
			WSendChannelThreadMessage func(ch string, thr string, msg string, v ...interface{}) ( robot.RetVal)
			WSendUserChannelMessage func(u string, ch string, msg string, v ...interface{}) ( robot.RetVal)
			WSendUserChannelThreadMessage func(u string, ch string, thr string, msg string, v ...interface{}) ( robot.RetVal)
			WSendUserMessage func(u string, msg string, v ...interface{}) ( robot.RetVal)
			WSetParameter func(a0 string, a1 string) ( bool)
			WSetWorkingDirectory func(a0 string) ( bool)
			WSpawnJob func(a0 string, a1 ...string) ( robot.RetVal)
			WThreaded func() ( robot.Robot)
			WUpdateDatum func(key string, locktoken string, datum interface{}) (ret robot.RetVal)

	}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) AddCommand(a0 string, a1 string) ( robot.RetVal) {return W.WAddCommand(a0, a1)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) AddJob(a0 string, a1 ...string) ( robot.RetVal) {return W.WAddJob(a0, a1...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) AddTask(a0 string, a1 ...string) ( robot.RetVal) {return W.WAddTask(a0, a1...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) CheckAdmin() ( bool) {return W.WCheckAdmin()
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) CheckinDatum(key string, locktoken string) () { W.WCheckinDatum(key, locktoken)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) CheckoutDatum(key string, datum interface{}, rw bool) (locktoken string, exists bool, ret robot.RetVal) {return W.WCheckoutDatum(key, datum, rw)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) Direct() ( robot.Robot) {return W.WDirect()
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) Elevate(a0 bool) ( bool) {return W.WElevate(a0)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) Email(subject string, messageBody *bytes.Buffer, html ...bool) (ret robot.RetVal) {return W.WEmail(subject, messageBody, html...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) EmailAddress(address string, subject string, messageBody *bytes.Buffer, html ...bool) (ret robot.RetVal) {return W.WEmailAddress(address, subject, messageBody, html...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) EmailUser(user string, subject string, messageBody *bytes.Buffer, html ...bool) (ret robot.RetVal) {return W.WEmailUser(user, subject, messageBody, html...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) Exclusive(tag string, queueTask bool) (success bool) {return W.WExclusive(tag, queueTask)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) FailCommand(a0 string, a1 string) ( robot.RetVal) {return W.WFailCommand(a0, a1)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) FailTask(a0 string, a1 ...string) ( robot.RetVal) {return W.WFailTask(a0, a1...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) FinalCommand(a0 string, a1 string) ( robot.RetVal) {return W.WFinalCommand(a0, a1)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) FinalTask(a0 string, a1 ...string) ( robot.RetVal) {return W.WFinalTask(a0, a1...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) Fixed() ( robot.Robot) {return W.WFixed()
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) GetBotAttribute(a string) ( *robot.AttrRet) {return W.WGetBotAttribute(a)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) GetMessage() ( *robot.Message) {return W.WGetMessage()
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) GetParameter(name string) ( string) {return W.WGetParameter(name)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) GetSenderAttribute(a string) ( *robot.AttrRet) {return W.WGetSenderAttribute(a)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) GetTaskConfig(cfgptr interface{}) ( robot.RetVal) {return W.WGetTaskConfig(cfgptr)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) GetUserAttribute(u string, a string) ( *robot.AttrRet) {return W.WGetUserAttribute(u, a)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) Log(l robot.LogLevel, m string, v ...interface{}) ( bool) {return W.WLog(l, m, v...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) MessageFormat(f robot.MessageFormat) ( robot.Robot) {return W.WMessageFormat(f)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) Pause(s float64) () { W.WPause(s)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) PromptForReply(regexID string, prompt string, v ...interface{}) ( string,  robot.RetVal) {return W.WPromptForReply(regexID, prompt, v...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) PromptThreadForReply(regexID string, prompt string, v ...interface{}) ( string,  robot.RetVal) {return W.WPromptThreadForReply(regexID, prompt, v...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) PromptUserChannelForReply(regexID string, user string, channel string, prompt string, v ...interface{}) ( string,  robot.RetVal) {return W.WPromptUserChannelForReply(regexID, user, channel, prompt, v...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) PromptUserChannelThreadForReply(regexID string, user string, channel string, thread string, prompt string, v ...interface{}) ( string,  robot.RetVal) {return W.WPromptUserChannelThreadForReply(regexID, user, channel, thread, prompt, v...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) PromptUserForReply(regexID string, user string, prompt string, v ...interface{}) ( string,  robot.RetVal) {return W.WPromptUserForReply(regexID, user, prompt, v...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) RandomInt(n int) ( int) {return W.WRandomInt(n)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) RandomString(s []string) ( string) {return W.WRandomString(s)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) Recall(key string, shared bool) ( string) {return W.WRecall(key, shared)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) Remember(key string, value string, shared bool) () { W.WRemember(key, value, shared)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) RememberContext(context string, value string) () { W.WRememberContext(context, value)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) RememberContextThread(context string, value string) () { W.WRememberContextThread(context, value)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) RememberThread(key string, value string, shared bool) () { W.WRememberThread(key, value, shared)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) Reply(msg string, v ...interface{}) ( robot.RetVal) {return W.WReply(msg, v...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) ReplyThread(msg string, v ...interface{}) ( robot.RetVal) {return W.WReplyThread(msg, v...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) Say(msg string, v ...interface{}) ( robot.RetVal) {return W.WSay(msg, v...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) SayThread(msg string, v ...interface{}) ( robot.RetVal) {return W.WSayThread(msg, v...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) SendChannelMessage(ch string, msg string, v ...interface{}) ( robot.RetVal) {return W.WSendChannelMessage(ch, msg, v...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) SendChannelThreadMessage(ch string, thr string, msg string, v ...interface{}) ( robot.RetVal) {return W.WSendChannelThreadMessage(ch, thr, msg, v...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) SendUserChannelMessage(u string, ch string, msg string, v ...interface{}) ( robot.RetVal) {return W.WSendUserChannelMessage(u, ch, msg, v...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) SendUserChannelThreadMessage(u string, ch string, thr string, msg string, v ...interface{}) ( robot.RetVal) {return W.WSendUserChannelThreadMessage(u, ch, thr, msg, v...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) SendUserMessage(u string, msg string, v ...interface{}) ( robot.RetVal) {return W.WSendUserMessage(u, msg, v...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) SetParameter(a0 string, a1 string) ( bool) {return W.WSetParameter(a0, a1)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) SetWorkingDirectory(a0 string) ( bool) {return W.WSetWorkingDirectory(a0)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) SpawnJob(a0 string, a1 ...string) ( robot.RetVal) {return W.WSpawnJob(a0, a1...)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) Threaded() ( robot.Robot) {return W.WThreaded()
			}
	func (W _github_com_lnxjedi_gopherbot_robot_Robot) UpdateDatum(key string, locktoken string, datum interface{}) (ret robot.RetVal) {return W.WUpdateDatum(key, locktoken, datum)
			}

// _github_com_lnxjedi_gopherbot_robot_SimpleBrain is an interface wrapper for SimpleBrain type
	type _github_com_lnxjedi_gopherbot_robot_SimpleBrain struct {
			IValue interface{}
			WDelete func(key string) ( error)
			WList func() (keys []string, err error)
			WRetrieve func(key string) (blob *[]byte, exists bool, err error)
			WStore func(key string, blob *[]byte) ( error)

	}
	func (W _github_com_lnxjedi_gopherbot_robot_SimpleBrain) Delete(key string) ( error) {return W.WDelete(key)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_SimpleBrain) List() (keys []string, err error) {return W.WList()
			}
	func (W _github_com_lnxjedi_gopherbot_robot_SimpleBrain) Retrieve(key string) (blob *[]byte, exists bool, err error) {return W.WRetrieve(key)
			}
	func (W _github_com_lnxjedi_gopherbot_robot_SimpleBrain) Store(key string, blob *[]byte) ( error) {return W.WStore(key, blob)
			}
