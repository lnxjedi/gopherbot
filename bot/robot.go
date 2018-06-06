package bot

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

// MessageFormat indicates how the connector should display the content of
// the message. One of Variable, Fixed or Raw
type MessageFormat int

// Outgoing message format, Variable or Fixed
const (
	Raw MessageFormat = iota // protocol native, zero value -> default if not specified
	Fixed
	Variable
)

// Connector protocols
type Protocol int

const (
	Slack Protocol = iota
	Terminal
	Test
)

//go:generate stringer -type=Protocol

// Generate String method with: go generate ./bot/

// Robot is created for each incoming message, in a separate goroutine that
// persists for the life of the message, until finally a plugin runs
// (or doesn't).
type Robot struct {
	User           string            // The user who sent the message; this can be modified for replying to an arbitrary user
	Channel        string            // The channel where the message was received, or "" for a direct message. This can be modified to send a message to an arbitrary channel.
	Protocol       Protocol          // slack, terminal, test, others; used for interpreting rawmsg or sending messages with Format = 'Raw'
	RawMsg         interface{}       // raw struct of message sent by connector; interpret based on protocol. For Slack this is a *slack.MessageEvent
	Format         MessageFormat     // The outgoing message format, one of Fixed or Variable
	callerID       string            // Pass the ID in for later identificaton of the calling plugin/job
	isCommand      bool              // Was the message directed at the robot, dm or by mention
	directMsg      bool              // if the message was sent by DM
	msg            string            // the message text sent
	bypassSecurity bool              // set for scheduled jobs, where user security restrictions don't apply
	elevated       bool              // set when required elevation succeeds
	environment    map[string]string // environment vars set for each job/plugin in the pipeline
	// NextJob, NextPlugin // TODO: create & use these data structures
}

type callerType int

const (
	plugin callerType = iota
	job
)

// a botCaller can be a plugin or a job, both capable of calling Robot methods
type botCaller struct {
	name          string         // name of job or plugin; unique by type, but job & plugin can share
	NameSpace     string         // callers that share namespace share long-term memories and environment vars; defaults to name if not otherwise set
	MaxHistories  int            // how many runs of this job/plugin to keep history for
	callerType    callerType     // plugin or job
	callerID      string         // 32-char random ID for identifying plugins/jobs in Robot method calls
	ReplyMatchers []InputMatcher // store this here for prompt*reply methods
	Disabled      bool
	reason        string // why this job/plugin is disabled
}

/* robot.go defines some convenience functions on struct Robot to
   simplify use by plugins. */

// CheckAdmin returns true if the user is a configured administrator of the
// robot. Should be used sparingly, when a single plugin has multiple commands,
// some which require admin. Otherwise the plugin should just configure
// RequireAdmin: true
func (r *Robot) CheckAdmin() bool {
	robot.RLock()
	defer robot.RUnlock()
	for _, adminUser := range robot.adminUsers {
		if r.User == adminUser {
			emit(AdminCheckPassed)
			return true
		}
	}
	emit(AdminCheckFailed)
	return false
}

// Elevate lets a plugin request elevation on the fly. When immediate = true,
// the elevator should always prompt for 2fa; otherwise a configured timeout
// should apply.
func (r *Robot) Elevate(immediate bool) bool {
	currentPlugins.RLock()
	plugins := currentPlugins.p
	plugin := plugins[currentPlugins.idMap[r.callerID]]
	currentPlugins.RUnlock()
	retval := r.elevate(plugins, plugin, immediate)
	if retval == Success {
		return true
	}
	return false
}

// Fixed is a deprecated convenience function for sending a message with fixed width
// font.
func (r *Robot) Fixed() *Robot {
	nr := *r
	nr.Format = Fixed
	return &nr
}

// MessageFormat returns a robot object with the given format, most likely for a
// plugin that will mostly use e.g. Variable format.
func (r *Robot) MessageFormat(f MessageFormat) *Robot {
	r.Format = f
	return r
}

// Direct is a convenience function for initiating a DM conversation with a
// user. Created initially so a plugin could prompt for a password in a DM.
func (r *Robot) Direct() *Robot {
	nr := *r
	nr.Channel = ""
	return &nr
}

// Pause is a convenience function to pause some fractional number of seconds.
func (r *Robot) Pause(s float64) {
	ms := time.Duration(s * float64(1000))
	time.Sleep(ms * time.Millisecond)
}

// RandomString is a convenience function for returning a random string
// from a slice of strings, so that replies can vary.
func (r *Robot) RandomString(s []string) string {
	l := len(s)
	if l == 0 {
		return ""
	}
	return s[random.Intn(l)]
}

// RandomInt uses the robot's seeded random to return a random int 0 <= retval < n
func (r *Robot) RandomInt(n int) int {
	return random.Intn(n)
}

// GetBotAttribute returns an attribute of the robot or "" if unknown.
// Current attributes:
// name, alias, fullName, contact
func (r *Robot) GetBotAttribute(a string) *AttrRet {
	a = strings.ToLower(a)
	robot.RLock()
	defer robot.RUnlock()
	ret := Ok
	var attr string
	switch a {
	case "name":
		attr = robot.name
	case "fullname", "realname":
		attr = robot.fullName
	case "alias":
		attr = string(robot.alias)
	case "email":
		attr = robot.email
	case "contact", "admin", "admincontact":
		attr = robot.adminContact
	case "protocol":
		attr = r.Protocol.String()
	default:
		ret = AttributeNotFound
	}
	return &AttrRet{attr, ret}
}

// GetUserAttribute returns a AttrRet with
// - The string Attribute of a user, or "" if unknown/error
// - A RetVal which is one of Ok, UserNotFound, AttributeNotFound
// Current attributes:
// name(handle), fullName, email, firstName, lastName, phone, internalID
// TODO: supplement data with gopherbot.json user's table
func (r *Robot) GetUserAttribute(u, a string) *AttrRet {
	a = strings.ToLower(a)
	attr, ret := robot.GetProtocolUserAttribute(u, a)
	return &AttrRet{attr, ret}
}

// messageHeard sends a typing notification
func (r *Robot) messageHeard() {
	robot.MessageHeard(r.User, r.Channel)
}

// GetSenderAttribute returns a AttrRet with
// - The string Attribute of the sender, or "" if unknown/error
// - A RetVal which is one of Ok, UserNotFound, AttributeNotFound
// Current attributes:
// name(handle), fullName, email, firstName, lastName, phone, internalID
// TODO: supplement data with gopherbot.json user's table
func (r *Robot) GetSenderAttribute(a string) *AttrRet {
	a = strings.ToLower(a)
	switch a {
	case "name", "username", "handle", "user", "user name":
		return &AttrRet{r.User, Ok}
	default:
		attr, ret := robot.GetProtocolUserAttribute(r.User, a)
		return &AttrRet{attr, ret}
	}
}

/*

GetPluginConfig sets a struct pointer to point to a config struct populated
from configuration when plugins were loaded. To use, a plugin should define
a struct for it's configuration data, e.g.:

	type pConf struct {
		Username, Password string
	}

In conf/plugins/<pluginname>.yaml, you would add a Config: stanza, e.g.:

	Config:
	  Username: foo
	  Password: bar

When registering the plugin, you pass a pointer to an empty config template, which the
robot will use to populate a struct when configuration is loaded:

	func init() {
		bot.RegisterPlugin("memes", bot.PluginHandler{
			DefaultConfig: defaultConfig, // yaml string providing default configuration
			Handler:       plugfunc, // callback function
			Config:        &pConf{}, // pointer to empty config struct
		})
	}

Then, to get a current copy of configuration when the plugin runs, define a struct pointer
and call GetPluginConfig with a double-pointer:

	var c *pConf
	r.GetPluginConfig(&c)

... And voila! *pConf is populated with the contents from the configured Config: stanza
*/
func (r *Robot) GetPluginConfig(dptr interface{}) RetVal {
	plugin := currentPlugins.getPluginByID(r.callerID)
	if plugin.config == nil {
		Log(Debug, fmt.Sprintf("Plugin \"%s\" called GetPluginConfig, but no config was found.", plugin.name))
		return NoConfigFound
	}
	tp := reflect.ValueOf(dptr)
	if tp.Kind() != reflect.Ptr {
		Log(Debug, fmt.Sprintf("Plugin \"%s\" called GetPluginConfig, but didn't pass a double-pointer to a struct", plugin.name))
		return InvalidDblPtr
	}
	p := reflect.Indirect(tp)
	if p.Kind() != reflect.Ptr {
		Log(Debug, fmt.Sprintf("Plugin \"%s\" called GetPluginConfig, but didn't pass a double-pointer to a struct", plugin.name))
		return InvalidDblPtr
	}
	if p.Type() != reflect.ValueOf(plugin.config).Type() {
		Log(Debug, fmt.Sprintf("Plugin \"%s\" called GetPluginConfig with an invalid double-pointer", plugin.name))
		return InvalidCfgStruct
	}
	p.Set(reflect.ValueOf(plugin.config))
	return Ok
}

// Log logs a message to the robot's log file (or stderr) if the level
// is lower than or equal to the robot's current log level
func (r *Robot) Log(l LogLevel, v ...interface{}) {
	Log(l, v...)
}

// SendChannelMessage lets a plugin easily send a message to an arbitrary
// channel. Use Robot.Fixed().SendChannelMessage(...) for fixed-width
// font.
func (r *Robot) SendChannelMessage(channel, msg string) RetVal {
	return robot.SendProtocolChannelMessage(channel, msg, r.Format)
}

// SendUserChannelMessage lets a plugin easily send a message directed to
// a specific user in a specific channel without fiddling with the robot
// object. Use Robot.Fixed().SencChannelMessage(...) for fixed-width
// font.
func (r *Robot) SendUserChannelMessage(user, channel, msg string) RetVal {
	return robot.SendProtocolUserChannelMessage(user, channel, msg, r.Format)
}

// SendUserMessage lets a plugin easily send a DM to a user. If a DM
// isn't possible, the connector should message the user in a channel.
func (r *Robot) SendUserMessage(user, msg string) RetVal {
	return robot.SendProtocolUserMessage(user, msg, r.Format)
}

// Reply directs a message to the user
func (r *Robot) Reply(msg string) RetVal {
	if r.Channel == "" {
		return robot.SendProtocolUserMessage(r.User, msg, r.Format)
	}
	return robot.SendProtocolUserChannelMessage(r.User, r.Channel, msg, r.Format)
}

// Say just sends a message to the user or channel
func (r *Robot) Say(msg string) RetVal {
	if r.Channel == "" {
		return robot.SendProtocolUserMessage(r.User, msg, r.Format)
	}
	return robot.SendProtocolChannelMessage(r.Channel, msg, r.Format)
}
