package bot

import "github.com/lnxjedi/gopherbot/robot"

const technicalAuthError = "Sorry, authorization failed due to a problem with the authorization plugin"
const configAuthError = "Sorry, authorization failed due to a configuration error"

// Check for a configured Authorizer and check authorization
func (r Robot) checkAuthorization(w *worker, t interface{}, command string, args ...string) (retval robot.TaskRetVal) {
	task, plugin, _ := getTask(t)
	isPlugin := plugin != nil
	if isPlugin {
		if !(plugin.AuthorizeAllCommands || len(plugin.AuthorizedCommands) > 0) {
			// This plugin requires no authorization
			if task.Authorizer != "" {
				Log(robot.Audit, "Plugin '%s' configured an authorizer, but has no commands requiring authorization", task.name)
				r.Say(configAuthError)
				return robot.ConfigurationError
			}
			return robot.Success
		} else if !plugin.AuthorizeAllCommands {
			authRequired := false
			for _, i := range plugin.AuthorizedCommands {
				if command == i {
					authRequired = true
					break
				}
			}
			if !authRequired {
				return robot.Success
			}
		}
	} else {
		// Jobs don't have commands; only check authorization if an Authorizer
		// is explicitly set.
		if len(task.Authorizer) == 0 {
			return robot.Success
		}
	}
	defaultAuthorizer := r.cfg.defaultAuthorizer
	if isPlugin && task.Authorizer == "" && defaultAuthorizer == "" {
		Log(robot.Audit, "Plugin '%s' requires authorization for command '%s', but no authorizer configured", task.name, command)
		r.Say(configAuthError)
		emit(AuthNoRunMisconfigured)
		return robot.ConfigurationError
	}
	authorizer := defaultAuthorizer
	if task.Authorizer != "" {
		authorizer = task.Authorizer
	}
	authTask := r.tasks.getTaskByName(authorizer)
	if authTask == nil {
		return robot.ConfigurationError
	}
	_, authPlug, _ := getTask(authTask)
	if authPlug != nil {
		args = append([]string{task.name, task.AuthRequire, command}, args...)
		_, authRet := w.callTask(authPlug, "authorize", args...)
		w.currentTask = r.currentTask
		if authRet == robot.Success {
			Log(robot.Audit, "Authorization succeeded by authorizer '%s' for user '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", authPlug.name, r.User, command, task.name, r.Channel, task.AuthRequire)
			emit(AuthRanSuccess)
			return robot.Success
		}
		if authRet == robot.Fail {
			Log(robot.Audit, "Authorization FAILED by authorizer '%s' for user '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", authPlug.name, r.User, command, task.name, r.Channel, task.AuthRequire)
			r.Say("Sorry, you're not authorized for that command")
			emit(AuthRanFail)
			return robot.Fail
		}
		if authRet == robot.MechanismFail {
			Log(robot.Audit, "Auth plugin '%s' mechanism failure while authenticating user '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", authPlug.name, r.User, command, task.name, r.Channel, task.AuthRequire)
			r.Say(technicalAuthError)
			emit(AuthRanMechanismFailed)
			return robot.MechanismFail
		}
		if authRet == robot.Normal {
			Log(robot.Audit, "Auth plugin '%s' returned 'Normal' (%d) instead of 'Success' (%d), failing auth in '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", authPlug.name, robot.Normal, robot.Success, r.User, command, task.name, r.Channel, task.AuthRequire)
			r.Say(technicalAuthError)
			emit(AuthRanFailNormal)
			return robot.MechanismFail
		}
		Log(robot.Audit, "Auth plugin '%s' exit code %s, failing auth while authenticating user '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", authPlug.name, authRet, r.User, command, task.name, r.Channel, task.AuthRequire)
		r.Say(technicalAuthError)
		emit(AuthRanFailOther)
		return robot.MechanismFail
	}
	Log(robot.Audit, "Auth plugin '%s' not found while authenticating user '%s' calling command '%s' for task '%s' in channel '%s'; AuthRequire: '%s'", task.Authorizer, r.User, command, task.name, r.Channel, task.AuthRequire)
	r.Say(technicalAuthError)
	emit(AuthNoRunNotFound)
	return robot.ConfigurationError
}
