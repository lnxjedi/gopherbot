package bot

import "fmt"

const technicalAuthError = "Sorry, authorization failed due to a problem with the authorization plugin"
const configAuthError = "Sorry, authorization failed due to a configuration error"

// Check for a configured Authorizer and check authorization
func (c *botContext) checkAuthorization(t interface{}, command string, args ...string) (retval TaskRetVal) {
	task, plugin, _ := getTask(t)
	r := c.makeRobot()
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
	} else {
		// Jobs don't have commands; only check authorization if an Authorizer
		// is explicitly set.
		if len(task.Authorizer) == 0 {
			return Success
		}
	}
	botCfg.RLock()
	defaultAuthorizer := botCfg.defaultAuthorizer
	botCfg.RUnlock()
	if isPlugin && task.Authorizer == "" && defaultAuthorizer == "" {
		Log(Audit, fmt.Sprintf("Plugin '%s' requires authorization for command '%s', but no authorizer configured", task.name, command))
		r.Say(configAuthError)
		emit(AuthNoRunMisconfigured)
		return ConfigurationError
	}
	authorizer := defaultAuthorizer
	if task.Authorizer != "" {
		authorizer = task.Authorizer
	}
	_, authPlug, _ := getTask(c.tasks.getTaskByName(authorizer))
	if authPlug != nil {
		args = append([]string{task.name, task.AuthRequire, command}, args...)
		_, authRet := c.callTask(authPlug, "authorize", args...)
		if authRet == Success {
			Log(Audit, fmt.Sprintf("Authorization succeeded by authorizer '%s' for user '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", authPlug.name, c.User, command, task.name, c.Channel, task.AuthRequire))
			emit(AuthRanSuccess)
			return Success
		}
		if authRet == Fail {
			Log(Audit, fmt.Sprintf("Authorization FAILED by authorizer '%s' for user '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", authPlug.name, c.User, command, task.name, c.Channel, task.AuthRequire))
			r.Say("Sorry, you're not authorized for that command")
			emit(AuthRanFail)
			return Fail
		}
		if authRet == MechanismFail {
			Log(Audit, fmt.Sprintf("Auth plugin '%s' mechanism failure while authenticating user '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", authPlug.name, c.User, command, task.name, c.Channel, task.AuthRequire))
			r.Say(technicalAuthError)
			emit(AuthRanMechanismFailed)
			return MechanismFail
		}
		if authRet == Normal {
			Log(Audit, fmt.Sprintf("Auth plugin '%s' returned 'Normal' (%d) instead of 'Success' (%d), failing auth in '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", authPlug.name, Normal, Success, c.User, command, task.name, c.Channel, task.AuthRequire))
			r.Say(technicalAuthError)
			emit(AuthRanFailNormal)
			return MechanismFail
		}
		Log(Audit, fmt.Sprintf("Auth plugin '%s' exit code %s, failing auth while authenticating user '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", authPlug.name, authRet, c.User, command, task.name, c.Channel, task.AuthRequire))
		r.Say(technicalAuthError)
		emit(AuthRanFailOther)
		return MechanismFail
	}
	Log(Audit, fmt.Sprintf("Auth plugin '%s' not found while authenticating user '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", task.Authorizer, c.User, command, task.name, c.Channel, task.AuthRequire))
	r.Say(technicalAuthError)
	emit(AuthNoRunNotFound)
	return ConfigurationError
}
