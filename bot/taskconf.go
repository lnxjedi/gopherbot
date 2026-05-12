package bot

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
	"gopkg.in/yaml.v3"
)

// loadTaskConfig() updates task/job/plugin configuration and namespaces/parametersets
// from robot.yaml and external configuration, then updates the
// globalTasks struct.
func loadTaskConfig(processed *configuration, preConnect bool) (*taskList, error) {
	newList := &taskList{
		t:             []interface{}{struct{}{}}, // initialize 0 to "nothing", for namespaces & parametersets only
		nameMap:       make(map[string]int),
		idMap:         make(map[string]int),
		nameSpaces:    make(map[string]ParameterSet),
		parameterSets: make(map[string]ParameterSet),
	}
	currentCfg.RLock()
	current := taskList{
		t:       currentCfg.t,
		nameMap: currentCfg.nameMap,
	}
	currentCfg.RUnlock()

	// Start with all the Go tasks, plugins and jobs
	for taskname := range taskHandlers {
		t := current.getTaskByName(taskname)
		newList.addTask(t)
	}

	for plugname := range pluginHandlers {
		t := current.getTaskByName(plugname)
		newList.addTask(t)
	}

	for jobname := range jobHandlers {
		t := current.getTaskByName(jobname)
		newList.addTask(t)
	}

	for _, ns := range processed.nsList {
		if _, ok := newList.nameMap[ns.Name]; ok {
			return newList, fmt.Errorf("NameSpace '%s' conflicts with another task/job/plugin/parameterset name", ns.Name)
		}
		newList.nameSpaces[ns.Name] = ParameterSet{
			name:        ns.Name,
			Description: ns.Description,
			Parameters:  ns.Parameters,
		}
		// The nameMap is the definitive list of all names, but namespaces don't correspond
		// to an actual task.
		newList.nameMap[ns.Name] = 0
	}

	for _, ps := range processed.psList {
		if _, ok := newList.nameMap[ps.Name]; ok {
			return newList, fmt.Errorf("ParameterSet '%s' conflicts with another task/job/plugin/parameterset name", ps.Name)
		}
		newList.parameterSets[ps.Name] = ParameterSet{
			name:        ps.Name,
			Description: ps.Description,
			Parameters:  ps.Parameters,
		}
		// The nameMap is the definitive list of all names, but parameter sets don't correspond
		// to an actual task.
		newList.nameMap[ps.Name] = 0
	}

	// Return disabled, error
	checkTaskSettings := func(ts TaskSettings, task *Task) (bool, error) {
		if ts.Disabled {
			task.Disabled = true
			task.reason = fmt.Sprintf("disabled in %s", robotConfigFileName)
			return true, nil
		}
		if len(ts.NameSpace) > 0 {
			if _, ok := newList.nameSpaces[ts.NameSpace]; !ok {
				return false, fmt.Errorf("configured NameSpace '%s' for task '%s' doesn't exist", ts.NameSpace, ts.Name)
			}
			task.NameSpace = ts.NameSpace
		}
		if len(ts.ParameterSets) > 0 {
			for _, set := range ts.ParameterSets {
				if _, ok := newList.parameterSets[set]; !ok {
					return false, fmt.Errorf("configured ParameterSet '%s' for task '%s' doesn't exist", set, ts.Name)
				}
			}
			task.ParameterSets = ts.ParameterSets
		}
		task.Description = ts.Description
		task.Parameters = ts.Parameters
		return false, nil
	}

	setupGoTask := func(ts TaskSettings, ttype pipeAddType) error {
		t := newList.getTaskByName(ts.Name)
		if t == nil {
			return fmt.Errorf("configuring Go task '%s' - no task found with that name", ts.Name)
		}
		task, plug, job := getTask(t)
		if (ttype == typePlugin && plug == nil) || (ttype == typeJob && job == nil) || task == nil {
			return fmt.Errorf("configuring Go task '%s' (type %s) - no task of that type registered with that name", ts.Name, ttype)
		}
		if (ttype == typePlugin) || (ttype == typeJob) {
			task.Privileged = *ts.Privileged
		}
		_, err := checkTaskSettings(ts, task)
		return err
	}

	// Get basic task configurations
	for _, ts := range processed.goTasks {
		if err := setupGoTask(ts, typeTask); err != nil {
			return newList, err
		}
	}
	for _, ts := range processed.goPlugins {
		if err := setupGoTask(ts, typePlugin); err != nil {
			return newList, err
		}
	}
	for _, ts := range processed.goJobs {
		if err := setupGoTask(ts, typeJob); err != nil {
			return newList, err
		}
	}

	addExternalTask := func(ts TaskSettings, ttype pipeAddType) (*Task, error) {
		if !identifierRe.MatchString(ts.Name) {
			return nil, fmt.Errorf("external task '%s' (type %s) doesn't match task name regex '%s'", ts.Name, ttype, identifierRe.String())
		}
		if ts.Name == "bot" {
			return nil, fmt.Errorf("illegal external task name 'bot' (type %s)", ts.Name)
		}
		if _, ok := newList.nameSpaces[ts.Name]; ok {
			return nil, fmt.Errorf("external task '%s' duplicates name of configured NameSpace", ts.Name)
		}
		if _, ok := newList.parameterSets[ts.Name]; ok {
			return nil, fmt.Errorf("external task '%s' duplicates name of configured ParameterSet", ts.Name)
		}
		if dupidx, ok := newList.nameMap[ts.Name]; ok {
			dupt := newList.t[dupidx]
			duptask, _, _ := getTask(dupt)
			if duptask.taskType == taskGo {
				return nil, fmt.Errorf("external task '%s' duplicates name of existing Go task/plugin/job", ts.Name)
			}
			return nil, fmt.Errorf("external task '%s' duplicates name of other external task/plugin/job", ts.Name)
		}
		task := &Task{
			name:        ts.Name,
			taskType:    taskExternal,
			Description: ts.Description,
			Parameters:  ts.Parameters,
			Homed:       ts.Homed,
			Privileged:  *ts.Privileged,
		}
		// Note that disabled external tasks are skipped in conf.go
		_, err := checkTaskSettings(ts, task)
		if err != nil {
			return nil, err
		}
		if len(ts.Path) == 0 {
			return nil, fmt.Errorf("zero-length path for external task '%s'", ts.Name)
		}
		var opath string
		var fileInfo fs.FileInfo
		if opath, fileInfo, err = getObjectPath(ts.Path); err != nil {
			return nil, fmt.Errorf("getting path '%s' for task '%s': %v", ts.Path, ts.Name, err)
		}
		if strings.HasSuffix(opath, ".go") && fileInfo.Mode()&0100 == 0 {
			if !task.Homed {
				task.Homed = true
				Log(robot.Warn, "Setting 'Homed: true' for external Go task '%s'", task.name)
			}
		}
		task.Path = ts.Path
		return task, nil
	}

	for _, script := range processed.externalPlugins {
		var task *Task
		var err error
		if task, err = addExternalTask(script, typePlugin); err != nil {
			return newList, err
		}
		p := &Plugin{
			Task: task,
		}
		newList.addTask(p)
	}

	for _, script := range processed.externalJobs {
		var task *Task
		var err error
		if task, err = addExternalTask(script, typeJob); err != nil {
			return newList, err
		}
		j := &Job{
			Task: task,
		}
		newList.addTask(j)
	}

	for _, script := range processed.externalTasks {
		var task *Task
		var err error
		if task, err = addExternalTask(script, typeTask); err != nil {
			return newList, err
		}
		newList.addTask(task)
	}

	// Load configuration for all valid tasks. Note that this is all being loaded
	// in to non-shared data structures that will replace current configuration
	// under lock at the end.
LoadLoop:
	for _, j := range newList.t[1:] {
		var plugin *Plugin
		var job *Job
		var task *Task
		var isPlugin, isJob bool
		switch t := j.(type) {
		case *Plugin:
			isPlugin = true
			plugin = t
			task = t.Task
			// Reset list of channels
			task.Channels = []string{}
		case *Job:
			isJob = true
			job = t
			task = t.Task
		// a bare task with no config to load
		default:
			continue
		}

		if task.Disabled {
			continue
		}
		tcfgdefault := make(map[string]interface{})
		tcfgload := make(map[string]json.RawMessage)
		if isPlugin {
			Log(robot.Info, "Loading configuration for plugin '%s', type %s", task.name, task.taskType)
		} else {
			Log(robot.Info, "Loading configuration for job '%s', type %s", task.name, task.taskType)
		}

		// Don't get plugin external configuration during preconnect,
		// since plugins may rely on stuff loaded by init jobs.
		if isPlugin && !preConnect {
			cfg, err := getDefCfg(j)
			if err != nil {
				msg := fmt.Sprintf("Getting default configuration for plugin, disabling: %v", err)
				Log(robot.Error, msg)
				task.Disabled = true
				task.reason = msg
				continue
			}
			if err := yaml.Unmarshal(*cfg, &tcfgdefault); err != nil {
				msg := fmt.Sprintf("Unmarshalling default configuration, disabling: %v", err)
				Log(robot.Error, "Problem unmarshalling plugin default config for '%s', disabling: %v", task.name, err)
				task.Disabled = true
				task.reason = msg
				continue
			}
		}
		// getConfigFile overlays the default config with configuration from the install path, then config path
		cpath := "jobs/"
		if isPlugin {
			cpath = "plugins/"
		}
		if err := getConfigFile(cpath+task.name+".yaml", false, tcfgload, tcfgdefault); err != nil {
			msg := fmt.Sprintf("Problem loading configuration file(s) for task '%s', disabling: %v", task.name, err)
			Log(robot.Error, msg)
			task.Disabled = true
			task.reason = msg
			continue
		}
		if disjson, ok := tcfgload["Disabled"]; ok {
			disabled := false
			if err := json.Unmarshal(disjson, &disabled); err != nil {
				msg := fmt.Sprintf("Problem unmarshalling value for 'Disabled' in plugin/job '%s', disabling: %v", task.name, err)
				Log(robot.Error, msg)
				task.Disabled = true
				task.reason = msg
				continue
			}
			if disabled {
				msg := fmt.Sprintf("Plugin/Job '%s' is disabled by configuration", task.name)
				Log(robot.Info, msg)
				task.Disabled = true
				task.reason = msg
				continue
			}
		}
		explicitAllChannels := false

		for key, value := range tcfgload {
			var strval string
			var intval int
			var boolval bool
			var sarrval []string
			var mval []InputMatcher
			var tval []JobTrigger
			var timeoutval TimeOutThresholds
			var val interface{}
			skip := false
			switch key {
			case "Elevator", "Authorizer", "AuthRequire", "NameSpace", "Channel":
				val = &strval
			case "KeepLogs":
				val = &intval
			case "TimeOuts":
				val = &timeoutval
			case "Disabled":
				skip = true
			case "AmbientMatchCommand", "AllChannels", "RequireAdmin", "AuthorizeAllCommands", "RequireAllCommandsPrivate", "RestrictPrivateChannels", "CatchAll", "MatchUnlisted", "Quiet":
				val = &boolval
			case "Channels", "ElevatedCommands", "ElevateImmediateCommands", "Users", "AuthorizedCommands", "AllowedPrivateCommands", "RequiredPrivateCommands", "AdminCommands", "ParameterSets", "CatchAllModes":
				val = &sarrval
			case "Commands", "ReplyMatchers", "MessageMatchers", "Arguments":
				val = &mval
			case "Triggers":
				val = &tval
			case "Config":
				skip = true
			case "Privileged":
				return newList, fmt.Errorf("task '%s' illegally specifies 'Privileged' outside of %s", task.name, robotConfigFileName)
			default:
				msg := fmt.Sprintf("Invalid configuration key for task '%s': %s - disabling", task.name, key)
				Log(robot.Error, msg)
				task.Disabled = true
				task.reason = msg
				continue LoadLoop
			}

			if !skip {
				if err := json.Unmarshal(value, val); err != nil {
					msg := fmt.Sprintf("Disabling plugin '%s' - error unmarshalling value '%s': %v", task.name, key, err)
					Log(robot.Error, msg)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				}
			}

			mismatch := false
			// Defaults
			switch key {
			// plugins can be scheduled, so Channel applies to both
			case "Channel":
				task.Channel = *(val.(*string))
			// Channels are only used for plugin visibility
			case "Channels":
				if isPlugin {
					task.Channels = *(val.(*[]string))
				} else {
					mismatch = true
				}
			case "AllChannels":
				task.AllChannels = *(val.(*bool))
				explicitAllChannels = true
			case "RequireAdmin":
				task.RequireAdmin = *(val.(*bool))
			case "AdminCommands":
				if isPlugin {
					plugin.AdminCommands = *(val.(*[]string))
				} else {
					mismatch = true
				}
			case "NameSpace":
				Log(robot.Error, "Task '%s' specifies NameSpace outside of %s, ignoring", robotConfigFileName)
			case "ParameterSets":
				Log(robot.Error, "Task '%s' specifies ParameterSets outside of %s, ignoring", robotConfigFileName)
			case "Elevator":
				task.Elevator = *(val.(*string))
			case "ElevatedCommands":
				if isPlugin {
					plugin.ElevatedCommands = *(val.(*[]string))
				} else {
					mismatch = true
				}
			case "ElevateImmediateCommands":
				if isPlugin {
					plugin.ElevateImmediateCommands = *(val.(*[]string))
				} else {
					mismatch = true
				}
			case "Users":
				task.Users = *(val.(*[]string))
			case "KeepLogs":
				if isPlugin {
					mismatch = true
				} else {
					job.KeepLogs = *(val.(*int))
				}
			case "Authorizer":
				task.Authorizer = *(val.(*string))
			case "AuthRequire":
				task.AuthRequire = *(val.(*string))
			case "AllowedPrivateCommands":
				if isPlugin {
					plugin.AllowedPrivateCommands = *(val.(*[]string))
				} else {
					mismatch = true
				}
			case "RequiredPrivateCommands":
				if isPlugin {
					plugin.RequiredPrivateCommands = *(val.(*[]string))
				} else {
					mismatch = true
				}
			case "CatchAllModes":
				if isPlugin {
					plugin.CatchAllModes = *(val.(*[]string))
				} else {
					mismatch = true
				}
			case "AuthorizedCommands":
				if isPlugin {
					plugin.AuthorizedCommands = *(val.(*[]string))
				} else {
					mismatch = true
				}
			case "AuthorizeAllCommands":
				if isPlugin {
					plugin.AuthorizeAllCommands = *(val.(*bool))
				} else {
					mismatch = true
				}
			case "RequireAllCommandsPrivate":
				if isPlugin {
					plugin.RequireAllCommandsPrivate = *(val.(*bool))
				} else {
					mismatch = true
				}
			case "RestrictPrivateChannels":
				if isPlugin {
					plugin.RestrictPrivateChannels = *(val.(*bool))
				} else {
					mismatch = true
				}
			case "Commands":
				if isPlugin {
					plugin.Commands = *(val.(*[]InputMatcher))
				} else {
					mismatch = true
				}
			case "ReplyMatchers":
				task.ReplyMatchers = *(val.(*[]InputMatcher))
			case "MessageMatchers":
				if isPlugin {
					plugin.MessageMatchers = *(val.(*[]InputMatcher))
				} else {
					mismatch = true
				}
			case "Arguments":
				if isPlugin {
					mismatch = true
				} else {
					job.Arguments = *(val.(*[]InputMatcher))
				}
			case "AmbientMatchCommand":
				if isPlugin {
					plugin.AmbientMatchCommand = *(val.(*bool))
				} else {
					mismatch = true
				}
			case "CatchAll":
				if isPlugin {
					plugin.CatchAll = *(val.(*bool))
				} else {
					mismatch = true
				}
			case "MatchUnlisted":
				if isPlugin {
					plugin.MatchUnlisted = *(val.(*bool))
				} else {
					mismatch = true
				}
			case "Quiet":
				if isPlugin {
					mismatch = true
				} else {
					job.Quiet = *(val.(*bool))
				}
			case "Triggers":
				if isPlugin {
					mismatch = true
				} else {
					job.Triggers = *(val.(*[]JobTrigger))
				}
			case "Config":
				task.Config = value
			case "TimeOuts":
				task.TimeOuts = *(val.(*TimeOutThresholds))
			}
			if mismatch {
				var msg string
				if isPlugin {
					msg = fmt.Sprintf("Disabling plugin '%s' - invalid configuration key: %s", task.name, key)
				} else {
					msg = fmt.Sprintf("Disabling job '%s' - invalid configuration key: %s", task.name, key)
				}
				Log(robot.Error, msg)
				task.Disabled = true
				task.reason = msg
				continue LoadLoop
			}
		}
		// End of reading configuration keys

		// Start sanity checking of configuration
		if err := validateTimeOutThresholds(fmt.Sprintf("task '%s' TimeOuts", task.name), task.TimeOuts); err != nil {
			msg := fmt.Sprintf("Disabling task '%s' - invalid TimeOuts: %v", task.name, err)
			Log(robot.Error, msg)
			task.Disabled = true
			task.reason = msg
			continue
		}
		if isJob || isPlugin {
			defaults := runtimeTimeOutThresholds{}
			if isJob {
				defaults = processed.timeOuts.Job
			} else {
				defaults = processed.timeOuts.Plugin
			}
			effective := resolveTimeOutThresholds(defaults, task.TimeOuts)
			if err := validateRuntimeTimeOutThresholds(fmt.Sprintf("task '%s' effective TimeOuts", task.name), effective); err != nil {
				msg := fmt.Sprintf("Disabling task '%s' - invalid effective TimeOuts: %v", task.name, err)
				Log(robot.Error, msg)
				task.Disabled = true
				task.reason = msg
				continue
			}
		}

		// Sanity checking / default for channel / channels
		if isJob && len(task.Channel) == 0 {
			task.Channel = processed.defaultJobChannel
		}
		if isPlugin {
			// Use bot default plugin channels if none defined, unless AllChannels requested.
			if len(task.Channels) == 0 {
				if len(processed.plugChannels) > 0 {
					if !task.AllChannels { // AllChannels = true is always explicit
						task.Channels = processed.plugChannels
					}
				} else { // no default channels specified
					if !explicitAllChannels { // if AllChannels wasn't explicitly configured, and no default channels, default to AllChannels = true
						task.AllChannels = true
					}
				}
			}
		}

		// Considering possible default channels, is the plugin visible anywhere?
		if isPlugin {
			if len(task.Channels) > 0 {
				msg := fmt.Sprintf("Plugin '%s' will be available in channels %q", task.name, task.Channels)
				Log(robot.Info, msg)
			} else {
				if !(pluginHasPrivatePolicy(plugin) || task.AllChannels) {
					msg := fmt.Sprintf("Plugin '%s' not visible in any channels or private command policy, disabling", task.name)
					Log(robot.Error, msg)
					task.Disabled = true
					task.reason = msg
					continue
				} else {
					Log(robot.Info, "Plugin '%s' has no channel restrictions configured; all channels: %t", task.name, task.AllChannels)
				}
			}
		} else {
			if len(task.Channel) == 0 {
				Log(robot.Error, "Job '%s' has no channel, and no DefaultJobChannel set, disabling", task.name)
				task.Disabled = true
				task.reason = "no channel set"
				continue
			} else {
				Log(robot.Info, "Job '%s' will run in channel '%s'", task.name, task.Channel)
			}
		}

		// Compile the regex's
		if isPlugin {
			for i := range plugin.Commands {
				command := &plugin.Commands[i]
				if err := compileInputMatcher(command, true); err != nil {
					msg := fmt.Sprintf("Disabling '%s', invalid command matcher for command '%s': %v", task.name, command.Command, err)
					Log(robot.Error, msg)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				}
			}
			for i := range plugin.MessageMatchers {
				// Note that full message regexes don't get the beginning and end anchors added - the individual plugin
				// will need to do this if necessary.
				message := &plugin.MessageMatchers[i]
				if strings.TrimSpace(message.SimpleMatcher) != "" {
					msg := fmt.Sprintf("Disabling '%s', SimpleMatcher is only supported for directed Commands", task.name)
					Log(robot.Error, msg)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				}
				re, err := regexp.Compile(message.Regex)
				if err != nil {
					msg := fmt.Sprintf("Disabling '%s', couldn't compile message regular expression '%s': %v", task.name, message.Regex, err)
					Log(robot.Error, msg)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				} else {
					message.re = re
				}
			}
		} else {
			for i := range job.Triggers {
				trigger := &job.Triggers[i]
				if len(trigger.User) == 0 || len(trigger.Channel) == 0 {
					msg := fmt.Sprintf("Disabling '%s', zero-length User or Channel for trigger #%d", task.name, i+1)
					Log(robot.Error, msg)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				}
				re, err := regexp.Compile(trigger.Regex)
				if err != nil {
					msg := fmt.Sprintf("Disabling '%s', couldn't compile trigger regular expression '%s': %v", task.name, trigger.Regex, err)
					Log(robot.Error, msg)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				} else {
					trigger.re = re
				}
			}
			for i := range job.Arguments {
				argument := &job.Arguments[i]
				label := argument.Label
				if stockRepliesRe.MatchString(label) {
					msg := fmt.Sprintf("Disabling '%s', invalid regex label '%s' starts with capital letter", task.name, label)
					Log(robot.Error, msg)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				}
				if err := compileInputMatcher(argument, false); err != nil {
					msg := fmt.Sprintf("Disabling '%s', invalid argument matcher '%s': %v", task.name, label, err)
					Log(robot.Error, msg)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				}
			}
		}
		for i := range task.ReplyMatchers {
			reply := &task.ReplyMatchers[i]
			label := reply.Label
			if stockRepliesRe.MatchString(label) {
				msg := fmt.Sprintf("Disabling '%s', invalid regex label '%s' starts with capital letter", task.name, label)
				Log(robot.Error, msg)
				task.Disabled = true
				task.reason = msg
				continue LoadLoop
			}
			if err := compileInputMatcher(reply, false); err != nil {
				msg := fmt.Sprintf("Skipping %s, invalid reply matcher '%s': %v", task.name, reply.Label, err)
				Log(robot.Error, msg)
				task.Disabled = true
				task.reason = msg
				continue LoadLoop
			}
		}

		// Make sure all security-related command lists resolve to actual
		// commands to guard against typos.
		if isPlugin {
			if len(plugin.CatchAllModes) > 0 {
				for _, mode := range plugin.CatchAllModes {
					switch strings.TrimSpace(strings.ToLower(mode)) {
					case "alias", "name", "direct", "hidden":
					default:
						msg := fmt.Sprintf("Disabling %s, invalid CatchAllModes value '%s' (expected alias, name, direct, or hidden)", task.name, mode)
						Log(robot.Error, msg)
						task.Disabled = true
						task.reason = msg
						continue LoadLoop
					}
				}
			}

			cmdlist := []struct {
				ctype string
				clist []string
			}{
				{"elevated", plugin.ElevatedCommands},
				{"elevate immediate", plugin.ElevateImmediateCommands},
				{"authorized", plugin.AuthorizedCommands},
				{"admin", plugin.AdminCommands},
				{"required private", plugin.RequiredPrivateCommands},
			}
			for _, cmd := range cmdlist {
				if len(cmd.clist) > 0 {
					for _, i := range cmd.clist {
						cmdfound := false
						for _, j := range plugin.Commands {
							if i == j.Command {
								cmdfound = true
								break
							}
						}
						if !cmdfound {
							for _, j := range plugin.MessageMatchers {
								if i == j.Command {
									cmdfound = true
									break
								}
							}
						}
						if !cmdfound {
							msg := fmt.Sprintf("Disabling %s, %s command %s didn't match a command from Commands or MessageMatchers", task.name, cmd.ctype, i)
							Log(robot.Error, msg)
							task.Disabled = true
							task.reason = msg
							continue LoadLoop
						}
					}
				}
			}

			authRequired := plugin.AuthorizeAllCommands || len(plugin.AuthorizedCommands) > 0
			localAuthorizerSet := strings.TrimSpace(task.Authorizer) != ""
			effectiveAuthorizer := strings.TrimSpace(task.Authorizer)
			if effectiveAuthorizer == "" {
				effectiveAuthorizer = strings.TrimSpace(processed.defaultAuthorizer)
			}

			// Fail fast for obvious authorization misconfiguration while preserving
			// runtime checks as defense in depth.
			if authRequired {
				if effectiveAuthorizer == "" {
					msg := fmt.Sprintf("Disabling %s, authorization is required but no Authorizer or DefaultAuthorizer is configured", task.name)
					Log(robot.Error, msg)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				}
				authTask := newList.getTaskByName(effectiveAuthorizer)
				if authTask == nil {
					msg := fmt.Sprintf("Disabling %s, configured authorizer '%s' was not found", task.name, effectiveAuthorizer)
					Log(robot.Error, msg)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				}
				_, authPlugin, _ := getTask(authTask)
				if authPlugin == nil {
					msg := fmt.Sprintf("Disabling %s, configured authorizer '%s' is not a plugin", task.name, effectiveAuthorizer)
					Log(robot.Error, msg)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				}
			} else if localAuthorizerSet {
				msg := fmt.Sprintf("Disabling %s, Authorizer is configured but no authorized commands are set", task.name)
				Log(robot.Error, msg)
				task.Disabled = true
				task.reason = msg
				continue LoadLoop
			}
		}

		Log(robot.Debug, "Configured task '%s'", task.name)
	}
	// End of configuration loading. All invalid tasks are disabled.

	return newList, nil
}
