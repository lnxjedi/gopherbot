package bot

import "fmt"

const technicalAuthError = "Sorry, authorization failed due to a problem with the authorization plugin"
const configAuthError = "Sorry, authorization failed due to a configuration error"

// Check for a configured Authorizer and check authorization
func (bot *botContext) checkAuthorization(t interface{}, command string, args ...string) (retval TaskRetVal) {
	task, plugin, _ := getTask(t)
	r := bot.makeRobot()
	isPlugin := plugin != nil
	if isPlugin {
		if !(plugin.AuthorizeAllCommands || len(plugin.AuthorizedCommands) > 0) {
			// This plugin requires no authorization
			if task.Authorizer != "" {
				Log(Audit, fmt.Sprintf("Plugin '%s' configured an authorizer, but has no commands requiring authorization", task.name))
				r.Say(configAuthError)
				return ConfigurationError
			}
			return Success
		} else if !plugin.AuthorizeAllCommands {
			authRequired := false
			for _, i := range plugin.AuthorizedCommands {
				if command == i {
					authRequired = true
					break
				}
			}
			if !authRequired {
				return Success
			}
		}
	}
	robot.RLock()
	defaultAuthorizer := robot.defaultAuthorizer
	robot.RUnlock()
	if isPlugin && task.Authorizer == "" && defaultAuthorizer == "" {
		Log(Audit, fmt.Sprintf("Plugin '%s' requires authorization for command '%s', but no authorizer configured", task.name, command))
		r.Say(configAuthError)
		emit(AuthNoRunMisconfigured)
		return ConfigurationError
	}
	authorizer := defaultAuthorizer
	if plugin.Authorizer != "" {
		authorizer = plugin.Authorizer
	}
	_, authPlug, _ := getTask(bot.tasks.getTaskByName(authorizer))
	if authPlug != nil {
		args = append([]string{task.name, task.AuthRequire, command}, args...)
		_, authRet := bot.callTask(authPlug, "authorize", args...)
		if authRet == Success {
			Log(Audit, fmt.Sprintf("Authorization succeeded by authorizer '%s' for user '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", authPlug.name, bot.User, command, task.name, bot.Channel, task.AuthRequire))
			emit(AuthRanSuccess)
			return Success
		}
		if authRet == Fail {
			Log(Audit, fmt.Sprintf("Authorization FAILED by authorizer '%s' for user '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", authPlug.name, bot.User, command, task.name, bot.Channel, task.AuthRequire))
			r.Say("Sorry, you're not authorized for that command in this channel")
			emit(AuthRanFail)
			return Fail
		}
		if authRet == MechanismFail {
			Log(Audit, fmt.Sprintf("Auth plugin '%s' mechanism failure while authenticating user '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", authPlug.name, bot.User, command, task.name, bot.Channel, task.AuthRequire))
			r.Say(technicalAuthError)
			emit(AuthRanMechanismFailed)
			return MechanismFail
		}
		if authRet == Normal {
			Log(Audit, fmt.Sprintf("Auth plugin '%s' returned 'Normal' (0) instead of 'Success' (1), failing auth in '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", authPlug.name, bot.User, command, task.name, bot.Channel, task.AuthRequire))
			r.Say(technicalAuthError)
			emit(AuthRanFailNormal)
			return MechanismFail
		}
		Log(Audit, fmt.Sprintf("Auth plugin '%s' exit code %s, failing auth while authenticating user '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", authPlug.name, authRet, bot.User, command, task.name, bot.Channel, task.AuthRequire))
		r.Say(technicalAuthError)
		emit(AuthRanFailOther)
		return MechanismFail
	}
	Log(Audit, fmt.Sprintf("Auth plugin '%s' not found while authenticating user '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", task.Authorizer, bot.User, command, task.name, bot.Channel, task.AuthRequire))
	r.Say(technicalAuthError)
	emit(AuthNoRunNotFound)
	return ConfigurationError
}
