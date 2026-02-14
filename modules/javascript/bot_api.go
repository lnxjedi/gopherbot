package javascript

import "github.com/lnxjedi/gopherbot/robot"

// BotAPI defines the robot methods used by the JavaScript runtime bridge.
// It is intentionally narrower than robot.Robot.
type BotAPI interface {
	CheckAdmin() bool
	Elevate(bool) bool
	GetBotAttribute(a string) *robot.AttrRet
	GetUserAttribute(u, a string) *robot.AttrRet
	GetSenderAttribute(a string) *robot.AttrRet
	GetTaskConfig(cfgptr interface{}) robot.RetVal
	GetParameter(name string) string
	Exclusive(tag string, queueTask bool) bool
	Fixed() BotAPI
	MessageFormat(f robot.MessageFormat) BotAPI
	Direct() BotAPI
	Threaded() BotAPI
	Log(l robot.LogLevel, m string, v ...interface{}) bool
	SendChannelMessage(ch, msg string, v ...interface{}) robot.RetVal
	SendChannelThreadMessage(ch, thr, msg string, v ...interface{}) robot.RetVal
	SendUserChannelMessage(u, ch, msg string, v ...interface{}) robot.RetVal
	SendProtocolUserChannelMessage(protocol, u, ch, msg string, v ...interface{}) robot.RetVal
	SendUserChannelThreadMessage(u, ch, thr, msg string, v ...interface{}) robot.RetVal
	SendUserMessage(u, msg string, v ...interface{}) robot.RetVal
	Reply(msg string, v ...interface{}) robot.RetVal
	ReplyThread(msg string, v ...interface{}) robot.RetVal
	Say(msg string, v ...interface{}) robot.RetVal
	SayThread(msg string, v ...interface{}) robot.RetVal
	RandomInt(n int) int
	RandomString(s []string) string
	Pause(s float64)
	PromptForReply(regexID string, prompt string, v ...interface{}) (string, robot.RetVal)
	PromptThreadForReply(regexID string, prompt string, v ...interface{}) (string, robot.RetVal)
	PromptUserForReply(regexID string, user string, prompt string, v ...interface{}) (string, robot.RetVal)
	PromptUserChannelForReply(regexID string, user, channel string, prompt string, v ...interface{}) (string, robot.RetVal)
	PromptUserChannelThreadForReply(regexID string, user, channel, thread string, prompt string, v ...interface{}) (string, robot.RetVal)
	CheckoutDatum(key string, datum interface{}, rw bool) (locktoken string, exists bool, ret robot.RetVal)
	CheckinDatum(key, locktoken string)
	UpdateDatum(key, locktoken string, datum interface{}) (ret robot.RetVal)
	Remember(key, value string, shared bool)
	RememberThread(key, value string, shared bool)
	RememberContext(context, value string)
	RememberContextThread(context, value string)
	Recall(key string, shared bool) string
	SpawnJob(string, ...string) robot.RetVal
	AddTask(string, ...string) robot.RetVal
	FinalTask(string, ...string) robot.RetVal
	FailTask(string, ...string) robot.RetVal
	AddJob(string, ...string) robot.RetVal
	AddCommand(string, string) robot.RetVal
	FinalCommand(string, string) robot.RetVal
	FailCommand(string, string) robot.RetVal
	SetParameter(string, string) bool
}
