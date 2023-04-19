package robot

import "bytes"

// Robot defines the methods exposed by gopherbot.bot Robot struct, for
// use by plugins/jobs/tasks. See bot/Robot for complete definitions.
type Robot interface {
	// Primarily plugin methods
	CheckAdmin() bool
	Elevate(bool) bool
	GetBotAttribute(a string) *AttrRet
	GetUserAttribute(u, a string) *AttrRet
	GetSenderAttribute(a string) *AttrRet
	GetTaskConfig(dptr interface{}) RetVal
	GetMessage() *Message
	GetParameter(name string) string
	Email(subject string, messageBody *bytes.Buffer, html ...bool) (ret RetVal)
	EmailUser(user, subject string, messageBody *bytes.Buffer, html ...bool) (ret RetVal)
	EmailAddress(address, subject string, messageBody *bytes.Buffer, html ...bool) (ret RetVal)
	Fixed() Robot
	MessageFormat(f MessageFormat) Robot
	Direct() Robot
	Threaded() Robot
	Log(l LogLevel, m string, v ...interface{}) bool
	SendChannelMessage(ch, msg string, v ...interface{}) RetVal
	SendChannelThreadMessage(ch, thr, msg string, v ...interface{}) RetVal
	SendUserChannelMessage(u, ch, msg string, v ...interface{}) RetVal
	SendUserChannelThreadMessage(u, ch, thr, msg string, v ...interface{}) RetVal
	SendUserMessage(u, msg string, v ...interface{}) RetVal
	Reply(msg string, v ...interface{}) RetVal
	ReplyThread(msg string, v ...interface{}) RetVal
	Say(msg string, v ...interface{}) RetVal
	SayThread(msg string, v ...interface{}) RetVal
	RandomInt(n int) int
	RandomString(s []string) string
	Pause(s float64)
	PromptForReply(regexID string, prompt string, v ...interface{}) (string, RetVal)
	PromptThreadForReply(regexID string, prompt string, v ...interface{}) (string, RetVal)
	PromptUserForReply(regexID string, user string, prompt string, v ...interface{}) (string, RetVal)
	PromptUserChannelForReply(regexID string, user, channel string, prompt string, v ...interface{}) (string, RetVal)
	PromptUserChannelThreadForReply(regexID string, user, channel, thread string, prompt string, v ...interface{}) (string, RetVal)
	CheckoutDatum(key string, datum interface{}, rw bool) (locktoken string, exists bool, ret RetVal)
	CheckinDatum(key, locktoken string)
	UpdateDatum(key, locktoken string, datum interface{}) (ret RetVal)
	Remember(key, value string, shared bool)
	RememberThread(key, value string, shared bool)
	RememberContext(context, value string)
	RememberContextThread(context, value string)
	Recall(key string, shared bool) string
	// Primarily job/pipeline methods
	GetRepoData() map[string]Repository
	ExtendNamespace(string, int) bool
	SpawnJob(string, ...string) RetVal
	AddTask(string, ...string) RetVal
	FinalTask(string, ...string) RetVal
	FailTask(string, ...string) RetVal
	AddJob(string, ...string) RetVal
	AddCommand(string, string) RetVal
	FinalCommand(string, string) RetVal
	FailCommand(string, string) RetVal
	// Go plugins only, for filesystem access to GOPHER_HOME
	RaisePriv(string)
	SetParameter(string, string) bool
	SetWorkingDirectory(string) bool
}
