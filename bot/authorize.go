package bot

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/lnxjedi/gopherbot/robot"
)

const technicalAuthError = "Sorry, authorization failed due to a problem with the authorization plugin"
const configAuthError = "Sorry, authorization failed due to a configuration error"
const authUserGroupsParamPrefix = "GopherbotAuthUsergroups"

var authUserGroupsRequestID uint64

func commandRequiresAuthorization(plugin *Plugin, command string) bool {
	if plugin == nil {
		return false
	}
	if plugin.AuthorizeAllCommands {
		return true
	}
	for _, authorized := range plugin.AuthorizedCommands {
		if command == authorized {
			return true
		}
	}
	return false
}

func effectiveAuthorizerName(task *Task, defaultAuthorizer string) string {
	if task == nil {
		return ""
	}
	if task.Authorizer != "" {
		return task.Authorizer
	}
	return defaultAuthorizer
}

func sanitizeParamToken(value string) string {
	if value == "" {
		return "authorizer"
	}
	var builder strings.Builder
	builder.Grow(len(value))
	for _, ch := range value {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			builder.WriteRune(ch)
		} else {
			builder.WriteByte('_')
		}
	}
	sanitized := builder.String()
	if sanitized == "" {
		return "authorizer"
	}
	return sanitized
}

func nextUserGroupsParamKey(authorizer string) string {
	seq := atomic.AddUint64(&authUserGroupsRequestID, 1)
	return fmt.Sprintf("%s_%s_%d", authUserGroupsParamPrefix, sanitizeParamToken(authorizer), seq)
}

func userHasRequiredGroup(groups map[string]struct{}, required string) bool {
	if len(required) == 0 || len(groups) == 0 {
		return false
	}
	if _, ok := groups[required]; ok {
		return true
	}
	_, ok := groups[strings.ToLower(required)]
	return ok
}

// getAuthorizerUserGroups asks an authorizer for user group membership using:
//
//	usergroups <username> <parameter-key>
//
// On Success, authorizers should call SetParameter(parameter-key, `["group1", ...]`).
// Any non-success return is treated as indeterminate group membership.
func (r Robot) getAuthorizerUserGroups(w *worker, authorizer, user string) (groups map[string]struct{}, known bool) {
	authorizer = strings.TrimSpace(authorizer)
	if authorizer == "" || strings.TrimSpace(user) == "" {
		return nil, false
	}
	authTask := r.tasks.getTaskByName(authorizer)
	if authTask == nil {
		return nil, false
	}
	_, authPlug, _ := getTask(authTask)
	if authPlug == nil {
		return nil, false
	}

	paramKey := nextUserGroupsParamKey(authorizer)

	w.Lock()
	c := w.pipeContext
	if c == nil {
		w.Unlock()
		return nil, false
	}
	delete(c.parameters, paramKey)
	w.Unlock()

	_, authRet := w.callTaskWithOptions(taskCallOptions{suppressEmit: true}, authPlug, "usergroups", user, paramKey)
	w.currentTask = r.currentTask
	if authRet != robot.Success {
		return nil, false
	}

	w.Lock()
	c = w.pipeContext
	if c == nil {
		w.Unlock()
		return nil, false
	}
	payload, ok := c.parameters[paramKey]
	delete(c.parameters, paramKey)
	w.Unlock()
	if !ok {
		return nil, false
	}
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return nil, false
	}

	var memberships []string
	if err := json.Unmarshal([]byte(payload), &memberships); err != nil {
		return nil, false
	}
	groups = make(map[string]struct{}, len(memberships)*2)
	for _, group := range memberships {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		groups[group] = struct{}{}
		groups[strings.ToLower(group)] = struct{}{}
	}
	return groups, true
}

// Check for a configured Authorizer and check authorization
func (r Robot) checkAuthorization(w *worker, t interface{}, command string, args ...string) (retval robot.TaskRetVal) {
	task, plugin, _ := getTask(t)
	isPlugin := plugin != nil
	if isPlugin {
		authConfigured := plugin.AuthorizeAllCommands || len(plugin.AuthorizedCommands) > 0
		if !authConfigured {
			// This plugin requires no authorization
			if task.Authorizer != "" {
				Log(robot.Audit, "Plugin '%s' configured an authorizer, but has no commands requiring authorization", task.name)
				r.Say(configAuthError)
				return robot.ConfigurationError
			}
			return robot.Success
		}
		if !commandRequiresAuthorization(plugin, command) {
			return robot.Success
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
	authorizer := effectiveAuthorizerName(task, defaultAuthorizer)
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
