package bot

import "fmt"

const technicalAuthError = "Sorry, authorization failed due to a problem with the authorization service"
const configAuthError = "Sorry, authorization failed due to a configuration error"

// Check for a configured Authorizer and check authorization
func (bot *Robot) checkAuthorization(plugins []*Plugin, plugin *Plugin, command string, args ...string) (retval PlugRetVal) {
	if !(plugin.AuthorizeAllCommands || len(plugin.AuthorizedCommands) > 0) {
		// This plugin requires no authorization
		if plugin.Authorizer != "" {
			Log(Error, fmt.Sprintf("Plugin \"%s\" configured an authorizer, but has no commands requiring authorization", plugin.name))
			bot.Say(configAuthError)
			return ConfigurationFail
		}
		return
	} else if !plugin.AuthorizeAllCommands {
		authRequired := false
		for _, i := range plugin.AuthorizedCommands {
			if command == i {
				authRequired = true
				break
			}
		}
		if !authRequired {
			return
		}
	}
	robot.RLock()
	defaultAuthorizer := robot.defaultAuthorizer
	robot.RUnlock()
	if plugin.Authorizer == "" && defaultAuthorizer == "" {
		Log(Error, fmt.Sprintf("Plugin \"%s\" requires authorization for command \"%s\", but no authorizer configured", plugin.name, command))
		bot.Say(configAuthError)
		return ConfigurationFail
	}
	authorizer := defaultAuthorizer
	if plugin.Authorizer != "" {
		authorizer = plugin.Authorizer
	}
	for _, authPlug := range plugins {
		if authorizer == authPlug.name {
			plugAllowed := false
			if authPlug.TrustAllPlugins {
				plugAllowed = true
			} else if len(authPlug.TrustedPlugins) > 0 {
				for _, allowed := range authPlug.TrustedPlugins {
					if plugin.name == allowed {
						plugAllowed = true
						break
					}
				}
			}
			if plugAllowed {
				if !pluginAvailable(bot.User, bot.Channel, authPlug) {
					Log(Error, fmt.Sprintf("Auth plugin \"%s\" not available while authenticating user \"%s\" calling command \"%s\" for plugin \"%s\" in channel \"%s\"; AuthRequire: \"%s\"", authPlug.name, bot.User, command, plugin.name, bot.Channel, plugin.AuthRequire))
					bot.Say(configAuthError)
					return ConfigurationFail
				}
				args = append([]string{plugin.name, "", plugin.AuthRequire, command}, args...)
				authRet := callPlugin(bot, authPlug, false, false, "authorize", args...)
				if authRet == Success {
					return Success
				}
				if authRet == Fail {
					Log(Warn, fmt.Sprintf("Authorization failed by authorizer \"%s\" for user \"%s\" calling command \"%s\" for plugin \"%s\" in channel \"%s\"; AuthRequire: \"%s\"", authPlug.name, bot.User, command, plugin.name, bot.Channel, plugin.AuthRequire))
					bot.Say("Sorry, you're not authorized for that command in this channel")
					return Fail
				}
				Log(Error, fmt.Sprintf("Auth plugin \"%s\" mechanism failure while authenticating user \"%s\" calling command \"%s\" for plugin \"%s\" in channel \"%s\"; AuthRequire: \"%s\"", authPlug.name, bot.User, command, plugin.name, bot.Channel, plugin.AuthRequire))
				bot.Say(technicalAuthError)
				return MechanismFail
			} else {
				Log(Error, fmt.Sprintf("Auth plugin \"%s\" not available to plugin \"%s\" while authenticating user \"%s\" calling command \"%s\" in channel \"%s\"; AuthRequire: \"%s\"", authPlug.name, plugin.name, bot.User, command, bot.Channel, plugin.AuthRequire))
				bot.Say(configAuthError)
				return ConfigurationFail
			}
		}
	}
	Log(Error, fmt.Sprintf("Auth plugin \"%s\" not found while authenticating user \"%s\" calling command \"%s\" for plugin \"%s\" in channel \"%s\"; AuthRequire: \"%s\"", plugin.Authorizer, bot.User, command, plugin.name, bot.Channel, plugin.AuthRequire))
	bot.Say(technicalAuthError)
	return ConfigurationFail
}
