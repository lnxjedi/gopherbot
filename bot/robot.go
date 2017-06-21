package bot

import (
	"fmt"
	"reflect"
	"time"
)

// MessageFormat indicates how the connector should display the content of
// the message. One of Variable, Fixed or Raw (To Be Implemented)
type MessageFormat int

// Outgoing message format, Variable or Fixed
const (
	Variable MessageFormat = iota // variable font width
	Fixed
	Raw
)

// Robot is passed to the plugin to enable convenience functions Say and Reply
type Robot struct {
	User     string        // The user who sent the message; this can be modified for replying to an arbitrary user
	Channel  string        // The channel where the message was received, or "" for a direct message. This can be modified to send a message to an arbitrary channel.
	Format   MessageFormat // The outgoing message format, one of Fixed or Variable
	pluginID string        // Pass the ID in for later identificaton of the plugin
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
			return true
		}
	}
	return false
}

// Elevate lets a plugin request elevation on the fly. When immediate = true,
// the elevator should always prompt for 2fa; otherwise a configured timeout
// should apply. Note that this can have unexpected side effects if the
// target of CallPlugin(...) calls Elevate.
func (r *Robot) Elevate(immediate bool) bool {
	currentPlugins.RLock()
	plugins := currentPlugins.p
	plugin := plugins[currentPlugins.idMap[r.pluginID]]
	currentPlugins.RUnlock()
	retval := r.elevate(plugins, plugin, immediate)
	if retval == Success {
		return true
	}
	return false
}

// Fixed is a convenience function for sending a message with fixed width
// font. e.g. r.Reply(xxx) replies in variable width font, but
// r.Fixed().Reply(xxx) replies in a fixed-width font.
func (r *Robot) Fixed() *Robot {
	nr := *r
	nr.Format = Fixed
	return &nr
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
	robot.RLock()
	defer robot.RUnlock()
	ret := Ok
	var attr string
	switch a {
	case "name":
		attr = robot.name
	case "fullName", "realName":
		attr = robot.fullName
	case "alias":
		attr = string(robot.alias)
	case "email":
		attr = robot.email
	case "contact", "admin", "adminContact":
		attr = robot.adminContact
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
	attr, ret := robot.GetProtocolUserAttribute(u, a)
	return &AttrRet{attr, ret}
}

// GetSenderAttribute returns a AttrRet with
// - The string Attribute of the sender, or "" if unknown/error
// - A RetVal which is one of Ok, UserNotFound, AttributeNotFound
// Current attributes:
// name(handle), fullName, email, firstName, lastName, phone, internalID
// TODO: supplement data with gopherbot.json user's table
func (r *Robot) GetSenderAttribute(a string) *AttrRet {
	attr, ret := robot.GetProtocolUserAttribute(r.User, a)
	return &AttrRet{attr, ret}
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
	plugin := currentPlugins.getPluginByID(r.pluginID)
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
// channel. Use Robot.Fixed().SencChannelMessage(...) for fixed-width
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
