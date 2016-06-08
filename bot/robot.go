package bot

import (
	"fmt"
	"reflect"
	"time"

	otp "github.com/dgryski/dgoogauth"
)

type MessageFormat int

// Outgoing message format, Variable or Fixed
const (
	Variable MessageFormat = iota // variable font width
	Fixed
)

// Robot is passed to the plugin to enable convenience functions Say and Reply
type Robot struct {
	User     string        // The user who sent the message; this can be modified for replying to an arbitrary user
	Channel  string        // The channel where the message was received, or "" for a direct message. This can be modified to send a message to an arbitrary channel.
	Format   MessageFormat // The outgoing message format, one of Fixed or Variable
	pluginID string        // Pass the ID in for later identificaton of the plugin
	*robot                 // Add a pointer to the robot for it's public methods, which includes the Connector provided by e.g. slack
}

/* robot.go defines some convenience functions on struct Robot to
   simplify use by plugins. */

// CheckAdmin returns true if the user is a configured administrator of the
// robot. Should be used sparingly, when a single plugin has multiple commands,
// some which require admin. Otherwise the plugin should just configure
// RequireAdmin: true
func (r Robot) CheckAdmin() bool {
	b := r.robot
	b.lock.RLock()
	defer b.lock.RUnlock()
	for _, adminUser := range b.adminUsers {
		if r.User == adminUser {
			return true
		}
	}
	return false
}

// CheckOTP returns true if the provided string is a valid OTP code for the
// user. See the builtInlaunchcodes.go plugin. Note that a plugin must be
// configured by the admin as Trusted to be able to use this function.
func (r Robot) CheckOTP(code string) (bool, BotRetVal) {
	b := r.robot
	b.lock.RLock()
	plugin := plugins[plugIDmap[r.pluginID]]
	trustedPlugin := plugin.Trusted
	plugName := plugin.Name
	b.lock.RUnlock()
	if !trustedPlugin {
		b.Log(Error, fmt.Sprintf("ALERT: Untrusted plugin \"%s\" called CheckOTP", plugName))
		return false, UntrustedPlugin
	}
	otpKey := "bot:OTP:" + r.User
	var userOTP otp.OTPConfig
	lock, exists, ret := r.checkoutDatum(otpKey, &userOTP, true)
	if ret != Ok {
		r.checkin(otpKey, lock)
		return false, NoUserOTP
	}
	if !exists {
		r.checkin(otpKey, lock)
		return false, ret
	}
	valid, err := userOTP.Authenticate(code)
	if err != nil {
		r.Log(Error, fmt.Errorf("Problem authenticating launch code for user %s: %v", r.User, err))
		r.checkin(otpKey, lock)
		return false, OTPError
	}
	ret = r.updateDatum(otpKey, lock, &userOTP)
	if ret != Ok {
		r.Log(Error, fmt.Errorf("Problem updating OTP for %s, failing", r.User))
		return false, ret
	}
	return valid, Ok
}

// Fixed is a convenience function for sending a message with fixed width
// font. e.g. r.Reply(xxx) replies in variable width font, but
// r.Fixed().Reply(xxx) replies in a fixed-width font.
func (r Robot) Fixed() Robot {
	r.Format = Fixed
	return r
}

// Pause is a convenience function to pause some fractional number of seconds.
func (r Robot) Pause(s float64) {
	ms := time.Duration(s * float64(1000))
	time.Sleep(ms * time.Millisecond)
}

// RandomString is a convenience function for returning a random string
// from a slice of strings, so that replies can vary.
func (r Robot) RandomString(s []string) string {
	l := len(s)
	if l == 0 {
		return ""
	}
	return s[random.Intn(l)]
}

// RandomInt uses the robot's seeded random to return a random int 0 <= retval < n
func (r Robot) RandomInt(n int) int {
	return random.Intn(n)
}

// GetBotAttribute returns an attribute of the robot or "" if unknown.
// Current attributes:
// name, alias, fullName, contact
func (r Robot) GetBotAttribute(a string) (attr string, ret BotRetVal) {
	b := r.robot
	b.lock.RLock()
	defer b.lock.RUnlock()
	switch a {
	case "name":
		attr = b.name
	case "fullName", "realName":
		attr = b.fullName
	case "alias":
		attr = string(b.alias)
	case "email":
		attr = b.email
	case "contact", "admin", "adminContact":
		attr = b.adminContact
	default:
		ret = AttributeNotFound
	}
	return
}

// GetUserAttribute returns an attribute of a user or "" if unknown/error
// Current attributes:
// name(handle), fullName, email, firstName, lastName, phone
// TODO: supplement data with gopherbot.json user's table
func (r Robot) GetUserAttribute(u, a string) (string, BotRetVal) {
	return r.GetProtocolUserAttribute(u, a)
}

// GetSenderAttribute returns an attribute of the sending user or "" if unknown/error
// Current attributes:
// name(handle), fullName, email, firstName, lastName, phone
// TODO: supplement data with gopherbot.json user's table
func (r Robot) GetSenderAttribute(a string) (string, BotRetVal) {
	return r.GetProtocolUserAttribute(r.User, a)
}

/* GetPluginConfig sets a struct pointer to point to a config struct populated
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
func (r Robot) GetPluginConfig(dptr interface{}) BotRetVal {
	b := r.robot
	b.lock.RLock()
	defer b.lock.RUnlock()
	plugin := plugins[plugIDmap[r.pluginID]]
	tp := reflect.ValueOf(dptr)
	if tp.Kind() != reflect.Ptr {
		b.Log(Debug, fmt.Sprintf("Plugin \"%s\" called GetPluginConfig, but didn't pass a double-pointer to a struct", plugin.Name))
		return InvalidDblPtr
	}
	p := reflect.Indirect(tp)
	if p.Kind() != reflect.Ptr {
		return InvalidDblPtr
	}
	if p.Type() != reflect.ValueOf(plugin.config).Type() {
		return InvalidCfgStruct
	}
	p.Set(reflect.ValueOf(plugin.config))
	return Ok
}

// SendChannelMessage lets a plugin easily send a message to an arbitrary
// channel. Use Robot.Fixed().SencChannelMessage(...) for fixed-width
// font.
func (r Robot) SendChannelMessage(channel, msg string) BotRetVal {
	return r.SendProtocolChannelMessage(channel, msg, r.Format)
}

// SendUserChannelMessage lets a plugin easily send a message directed to
// a specific user in a specific channel without fiddling with the robot
// object. Use Robot.Fixed().SencChannelMessage(...) for fixed-width
// font.
func (r Robot) SendUserChannelMessage(user, channel, msg string) BotRetVal {
	return r.SendProtocolUserChannelMessage(user, channel, msg, r.Format)
}

// SendUserMessage lets a plugin easily send a DM to a user. If a DM
// isn't possible, the connector should message the user in a channel.
func (r Robot) SendUserMessage(user, msg string) BotRetVal {
	return r.SendProtocolUserMessage(user, msg, r.Format)
}

// Reply directs a message to the user
func (r Robot) Reply(msg string) BotRetVal {
	if r.Channel == "" {
		return r.SendProtocolUserMessage(r.User, msg, r.Format)
	} else {
		return r.SendProtocolUserChannelMessage(r.User, r.Channel, msg, r.Format)
	}
}

// Say just sends a message to the user or channel
func (r Robot) Say(msg string) BotRetVal {
	if r.Channel == "" {
		return r.SendProtocolUserMessage(r.User, msg, r.Format)
	} else {
		return r.SendProtocolChannelMessage(r.Channel, msg, r.Format)
	}
}
