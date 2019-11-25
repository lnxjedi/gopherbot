package robot

import "bytes"

// Robot defines the methods exposed by gopherbot.bot Robot struct, for
// use by plugins.
type Robot interface {
	CheckAdmin() bool
	GetBotAttribute(a string) *AttrRet
	GetUserAttribute(u, a string) *AttrRet
	GetSenderAttribute(a string) *AttrRet
	GetTaskConfig(dptr interface{}) RetVal
	GetMessage() *Message
	GetSecret(name string) string
	Email(subject string, messageBody *bytes.Buffer, html ...bool) (ret RetVal)
	EmailUser(user, subject string, messageBody *bytes.Buffer, html ...bool) (ret RetVal)
	EmailAddress(address, subject string, messageBody *bytes.Buffer, html ...bool) (ret RetVal)
	Fixed() Robot
	MessageFormat(f MessageFormat) Robot
	Direct() Robot
	Log(l LogLevel, m string, v ...interface{}) bool
	SendChannelMessage(ch, msg string, v ...interface{}) RetVal
	SendUserChannelMessage(u, ch, msg string, v ...interface{}) RetVal
	SendUserMessage(u, msg string, v ...interface{}) RetVal
	Reply(msg string, v ...interface{}) RetVal
	Say(msg string, v ...interface{}) RetVal
	RandomInt(n int) int
	RandomString(s []string) string
	Pause(s float64)
	PromptForReply(regexID string, prompt string, v ...interface{}) (string, RetVal)
	PromptUserForReply(regexID string, user string, prompt string, v ...interface{}) (string, RetVal)
	PromptUserChannelForReply(regexID string, user string, channel string, prompt string, v ...interface{}) (string, RetVal)
	CheckoutDatum(key string, datum interface{}, rw bool) (locktoken string, exists bool, ret RetVal)
	CheckinDatum(key, locktoken string)
	UpdateDatum(key, locktoken string, datum interface{}) (ret RetVal)
	Remember(key, value string)
	RememberContext(context, value string)
	Recall(key string) string
}
