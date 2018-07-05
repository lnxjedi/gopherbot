package bot

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"

	"github.com/ghodss/yaml"
)

// loadTaskConfig() loads the configuration for all the jobs/plugins from
// /jobs/<jobname>.yaml or /plugins/<pluginname>.yaml, assigns a taskID, and
// stores the resulting array in b.tasks. Bad tasks are skipped and logged.
// Task configuration is initially loaded into temporary data structures,
// then stored in the bot package under the global bot lock.
func (r *botContext) loadTaskConfig() {
	taskIndexByID := make(map[string]int)
	taskIndexByName := make(map[string]int)
	nameSpaceSet := make(map[string]struct{})
	tlist := make([]interface{}, 0, 14)

	// Copy some data from the bot under read lock, including external plugins
	robot.RLock()
	defaultAllowDirect := robot.defaultAllowDirect
	// copy the list of default channels (for plugins only)
	pchan := robot.plugChannels
	jdefchan := robot.defaultJobChannel
	externalPlugins := robot.externalPlugins
	externalJobs := robot.externalJobs
	robot.RUnlock() // we're done with bot data 'til the end

	i := 0

	for plugname := range pluginHandlers {
		plugin := &botPlugin{
			botTask: &botTask{
				name:     plugname,
				taskType: taskGo,
				taskID:   getTaskID(plugname),
			},
		}
		tlist = append(tlist, plugin)
		taskIndexByID[plugin.botTask.taskID] = i
		taskIndexByName[plugin.botTask.name] = i
		i++
	}

	// Initial load of plugins
	for index, script := range externalPlugins {
		if !identifierRe.MatchString(script.Name) {
			Log(Error, fmt.Sprintf("Plugin name: '%s', index: %d doesn't match task name regex '%s', skipping", script.Name, index+1, identifierRe.String()))
			continue
		}
		if script.Name == "bot" {
			Log(Error, "Illegal task name: bot - skipping")
			continue
		}
		if _, ok := taskIndexByName[script.Name]; ok {
			msg := fmt.Sprintf("External plugin index: #%d, name: '%s' duplicates name of builtIn or Go plugin, skipping", index, script.Name)
			Log(Error, msg)
			r.debug(msg, false)
			continue
		}
		task := &botTask{
			name:     script.Name,
			taskType: taskExternal,
			taskID:   getTaskID(script.Name),
			Path:     script.Path,
		}
		p := &botPlugin{
			botTask: task,
		}
		tlist = append(tlist, p)
		taskIndexByID[task.taskID] = i
		taskIndexByName[task.name] = i
		i++
	}

	// Initial load of jobs
	for index, script := range externalJobs {
		if !identifierRe.MatchString(script.Name) {
			Log(Error, fmt.Sprintf("Job name: '%s', index: %d doesn't match task name regex '%s', skipping", script.Name, index+1, identifierRe.String()))
			continue
		}
		if script.Name == "bot" {
			Log(Error, "Illegal task name: bot - skipping")
			continue
		}
		if _, ok := taskIndexByName[script.Name]; ok {
			msg := fmt.Sprintf("External job index: #%d, name: '%s' duplicates name of builtIn or Go plugin, skipping", index, script.Name)
			Log(Error, msg)
			r.debug(msg, false)
			continue
		}
		task := &botTask{
			name:        script.Name,
			taskType:    taskExternal,
			taskID:      getTaskID(script.Name),
			Description: script.Description,
		}
		j := &botJob{
			botTask: task,
		}
		tlist = append(tlist, j)
		taskIndexByID[task.taskID] = i
		taskIndexByName[task.name] = i
		i++
	}

	// Load configuration for all valid tasks. Note that this is all being loaded
	// in to non-shared data structures that will replace current configuration
	// under lock at the end.
LoadLoop:
	for _, j := range tlist {
		var plugin *botPlugin
		var job *botJob
		var task *botTask
		var isPlugin bool
		switch t := j.(type) {
		case *botPlugin:
			isPlugin = true
			plugin = t
			task = t.botTask
		case *botJob:
			job = t
			task = t.botTask
		}

		if task.Disabled {
			continue
		}
		tcfgload := make(map[string]json.RawMessage)
		if isPlugin {
			Log(Info, fmt.Sprintf("Loading configuration for plugin '%s', type %d", task.name, plugin.taskType))
		} else {
			Log(Info, fmt.Sprintf("Loading configuration for job '%s'", task.name))
		}

		if isPlugin {
			if plugin.taskType == taskExternal {
				// External plugins spit their default config to stdout when called with command="configure"
				cfg, err := getExtDefCfg(task)
				if err != nil {
					msg := fmt.Sprintf("Error getting default configuration for external plugin, disabling: %v", err)
					Log(Error, msg)
					r.debug(msg, false)
					task.Disabled = true
					task.reason = msg
					continue
				}
				if len(*cfg) > 0 {
					r.debug(fmt.Sprintf("Loaded default config from the plugin, size: %d", len(*cfg)), false)
				} else {
					r.debug("Unable to obtain default config from plugin, command 'configure' returned no content", false)
				}
				if err := yaml.Unmarshal(*cfg, &tcfgload); err != nil {
					msg := fmt.Sprintf("Error unmarshalling default configuration, disabling: %v", err)
					Log(Error, fmt.Errorf("Problem unmarshalling plugin default config for '%s', disabling: %v", task.name, err))
					r.debug(msg, false)
					task.Disabled = true
					task.reason = msg
					continue
				}
			} else {
				if err := yaml.Unmarshal([]byte(pluginHandlers[task.name].DefaultConfig), &tcfgload); err != nil {
					msg := fmt.Sprintf("Error unmarshalling default configuration, disabling: %v", err)
					Log(Error, fmt.Errorf("Problem unmarshalling plugin default config for '%s', disabling: %v", task.name, err))
					r.debug(msg, false)
					task.Disabled = true
					task.reason = msg
					continue
				}
			}
		}
		// getConfigFile overlays the default config with configuration from the install path, then config path
		cpath := "jobs/"
		if isPlugin {
			cpath = "plugins/"
		}
		if err := r.getConfigFile(cpath+task.name+".yaml", task.taskID, false, tcfgload); err != nil {
			msg := fmt.Sprintf("Problem loading configuration file(s) for task '%s', disabling: %v", task.name, err)
			Log(Error, msg)
			r.debug(msg, false)
			task.Disabled = true
			task.reason = msg
			continue
		}
		if disjson, ok := tcfgload["Disabled"]; ok {
			disabled := false
			if err := json.Unmarshal(disjson, &disabled); err != nil {
				msg := fmt.Sprintf("Problem unmarshalling value for 'Disabled' in plugin '%s', disabling: %v", task.name, err)
				Log(Error, msg)
				r.debug(msg, false)
				task.Disabled = true
				task.reason = msg
				continue
			}
			if disabled {
				msg := fmt.Sprintf("Plugin '%s' is disabled by configuration", task.name)
				Log(Info, msg)
				r.debug(msg, false)
				task.Disabled = true
				task.reason = msg
				continue
			}
		}
		// Boolean false values can be explicitly false, or default to false
		// when not specified. In some cases that matters.
		explicitAllChannels := false
		explicitAllowDirect := false

		for key, value := range tcfgload {
			var strval string
			var intval int
			var boolval bool
			var sarrval []string
			var hval []PluginHelp
			var mval []InputMatcher
			var pval []parameter
			var val interface{}
			skip := false
			switch key {
			case "Description", "Elevator", "Authorizer", "AuthRequire", "NameSpace", "Channel", "User", "Path":
				val = &strval
			case "Parameters":
				val = &pval
			case "HistoryLogs":
				val = &intval
			case "Disabled", "AllowDirect", "DirectOnly", "DenyDirect", "AllChannels", "RequireAdmin", "AuthorizeAllCommands", "CatchAll", "PrivateNameSpace", "Verbose":
				val = &boolval
			case "Channels", "ElevatedCommands", "ElevateImmediateCommands", "Users", "AuthorizedCommands", "AdminCommands", "RequiredParameters":
				val = &sarrval
			case "Help":
				val = &hval
			case "CommandMatchers", "ReplyMatchers", "MessageMatchers", "Triggers":
				val = &mval
			case "Config":
				skip = true
			default:
				msg := fmt.Sprintf("Invalid configuration key for task '%s': %s - disabling", task.name, key)
				Log(Error, msg)
				r.debug(msg, false)
				task.Disabled = true
				task.reason = msg
				continue LoadLoop
			}

			if !skip {
				if err := json.Unmarshal(value, val); err != nil {
					msg := fmt.Sprintf("Disabling plugin '%s' - error unmarshalling value '%s': %v", task.name, key, err)
					Log(Error, msg)
					r.debug(msg, false)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				}
			}

			mismatch := false
			// Defaults
			if isPlugin {
				task.PrivateNameSpace = true
			}
			switch key {
			case "AllowDirect":
				task.AllowDirect = *(val.(*bool))
				explicitAllowDirect = true
			case "DirectOnly":
				task.DirectOnly = *(val.(*bool))
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
			case "Description":
				if isPlugin {
					task.Description = *(val.(*string))
				} else {
					if len(task.Description) == 0 {
						task.Description = *(val.(*string))
					}
				}
			case "NameSpace":
				task.NameSpace = *(val.(*string))
				if !identifierRe.MatchString(task.NameSpace) {
					Log(Error, fmt.Sprintf("Task '%s' has invalid NameSpace '%s'; doesn't match regex '%s', ignoring", task.name, task.NameSpace, identifierRe.String()))
					task.NameSpace = ""
				}
				if task.NameSpace == "bot" {
					Log(Error, fmt.Sprintf("Task '%s' has illegal NameSpace 'bot', ignoring", task.name))
					task.NameSpace = ""
				}
			case "PrivateNameSpace":
				task.PrivateNameSpace = *(val.(*bool))
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
			case "HistoryLogs":
				task.HistoryLogs = *(val.(*int))
			case "Authorizer":
				task.Authorizer = *(val.(*string))
			case "AuthRequire":
				task.AuthRequire = *(val.(*string))
			case "User":
				task.User = *(val.(*string))
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
			case "Help":
				if isPlugin {
					plugin.Help = *(val.(*[]PluginHelp))
				} else {
					mismatch = true
				}
			case "CommandMatchers":
				if isPlugin {
					plugin.CommandMatchers = *(val.(*[]InputMatcher))
				} else {
					mismatch = true
				}
			case "ReplyMatchers":
				if isPlugin {
					task.ReplyMatchers = *(val.(*[]InputMatcher))
				} else {
					mismatch = true
				}
			case "MessageMatchers":
				if isPlugin {
					plugin.MessageMatchers = *(val.(*[]InputMatcher))
				} else {
					mismatch = true
				}
			case "CatchAll":
				if isPlugin {
					plugin.CatchAll = *(val.(*bool))
				} else {
					mismatch = true
				}
			case "Verbose":
				if isPlugin {
					mismatch = true
				} else {
					job.Verbose = *(val.(*bool))
				}
			case "Triggers":
				if isPlugin {
					mismatch = true
				} else {
					job.Triggers = *(val.(*[]InputMatcher))
				}
			case "Parameters":
				if isPlugin {
					mismatch = true
				} else {
					job.Parameters = *(val.(*[]parameter))
				}
			case "Path":
				if isPlugin {
					mismatch = true
				} else {
					task.Path = *(val.(*string))
				}
			case "RequiredParameters":
				if isPlugin {
					mismatch = true
				} else {
					job.RequiredParameters = *(val.(*[]string))
				}
			case "Config":
				task.Config = value
			}
			if mismatch {
				var msg string
				if isPlugin {
					msg = fmt.Sprintf("Disabling plugin '%s' - invalid configuration key: %s", task.name, key)
				} else {
					msg = fmt.Sprintf("Disabling job '%s' - invalid configuration key: %s", task.name, key)
				}
				Log(Error, msg)
				r.debug(msg, false)
				task.Disabled = true
				task.reason = msg
				continue LoadLoop
			}
		}
		// End of reading configuration keys

		// Start sanity checking of configuration
		if len(task.Path) == 0 && task.taskType == taskExternal {
			msg := fmt.Sprintf("Task '%s' has zero-length path, disabling", task.name)
			Log(Error, msg)
			r.debug(msg, false)
			task.Disabled = true
			task.reason = msg
		}
		if task.DirectOnly {
			if explicitAllowDirect {
				if !task.AllowDirect {
					msg := fmt.Sprintf("Task '%s' has conflicting values for AllowDirect (false) and DirectOnly (true), disabling", task.name)
					Log(Error, msg)
					r.debug(msg, false)
					task.Disabled = true
					task.reason = msg
					continue
				}
			} else {
				Log(Debug, "DirectOnly specified without AllowDirect; setting AllowDirect = true")
				task.AllowDirect = true
				explicitAllowDirect = true
			}
		}
		if len(task.NameSpace) == 0 {
			task.NameSpace = task.name
		}
		nameSpaceSet[task.NameSpace] = struct{}{}

		if !explicitAllowDirect {
			task.AllowDirect = defaultAllowDirect
		}

		// Sanity checking / default for channel / channels
		if len(task.Channel) == 0 {
			task.Channel = jdefchan
		}
		if isPlugin {
			// Use bot default plugin channels if none defined, unless AllChannels requested.
			if len(task.Channels) == 0 {
				if len(pchan) > 0 {
					if !task.AllChannels { // AllChannels = true is always explicit
						task.Channels = pchan
					}
				} else { // no default channels specified
					if !explicitAllChannels { // if AllChannels wasn't explicitly configured, and no default channels, default to AllChannels = true
						task.AllChannels = true
					}
				}
			}
		} else {
			task.Channels = []string{task.Channel}
		}

		// Considering possible default channels, is the plugin visible anywhere?
		if len(task.Channels) > 0 {
			msg := fmt.Sprintf("Task '%s' will be available in channels %q", task.name, task.Channels)
			Log(Info, msg)
			r.debug(msg, false)
		} else {
			if !(task.AllowDirect || task.AllChannels) {
				msg := fmt.Sprintf("Task '%s' not visible in any channels or by direct message, disabling", task.name)
				Log(Error, msg)
				r.debug(msg, false)
				task.Disabled = true
				task.reason = msg
				continue
			} else {
				msg := fmt.Sprintf("Task '%s' has no channel restrictions configured; all channels: %t", task.name, task.AllChannels)
				Log(Info, msg)
				r.debug(msg, false)
			}
		}

		// Compile the regex's
		if isPlugin {
			for i := range plugin.CommandMatchers {
				command := &plugin.CommandMatchers[i]
				regex := massageRegexp(command.Regex)
				re, err := regexp.Compile(`^\s*` + regex + `\s*$`)
				if err != nil {
					msg := fmt.Sprintf("Disabling %s, couldn't compile command regular expression '%s': %v", task.name, regex, err)
					Log(Error, msg)
					r.debug(msg, false)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				} else {
					command.re = re
				}
			}
			for i := range plugin.MessageMatchers {
				// Note that full message regexes don't get the beginning and end anchors added - the individual plugin
				// will need to do this if necessary.
				message := &plugin.MessageMatchers[i]
				regex := massageRegexp(message.Regex)
				re, err := regexp.Compile(regex)
				if err != nil {
					msg := fmt.Sprintf("Skipping %s, couldn't compile message regular expression '%s': %v", task.name, regex, err)
					Log(Error, msg)
					r.debug(msg, false)
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
				regex := massageRegexp(trigger.Regex)
				re, err := regexp.Compile(`^\s*` + regex + `\s*$`)
				if err != nil {
					msg := fmt.Sprintf("Disabling %s, couldn't compile trigger regular expression '%s': %v", task.name, regex, err)
					Log(Error, msg)
					r.debug(msg, false)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				} else {
					trigger.re = re
				}
			}
		}
		for i := range task.ReplyMatchers {
			reply := &task.ReplyMatchers[i]
			regex := massageRegexp(reply.Regex)
			re, err := regexp.Compile(`^\s*` + regex + `\s*$`)
			if err != nil {
				msg := fmt.Sprintf("Skipping %s, couldn't compile reply regular expression '%s': %v", task.name, regex, err)
				Log(Error, msg)
				r.debug(msg, false)
				task.Disabled = true
				task.reason = msg
				continue LoadLoop
			} else {
				reply.re = re
			}
		}

		// Make sure all security-related command lists resolve to actual
		// commands to guard against typos.
		if isPlugin {
			cmdlist := []struct {
				ctype string
				clist []string
			}{
				{"elevated", plugin.ElevatedCommands},
				{"elevate immediate", plugin.ElevateImmediateCommands},
				{"authorized", plugin.AuthorizedCommands},
				{"admin", plugin.AdminCommands},
			}
			for _, cmd := range cmdlist {
				if len(cmd.clist) > 0 {
					for _, i := range cmd.clist {
						cmdfound := false
						for _, j := range plugin.CommandMatchers {
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
							msg := fmt.Sprintf("Disabling %s, %s command %s didn't match a command from CommandMatchers or MessageMatchers", task.name, cmd.ctype, i)
							Log(Error, msg)
							r.debug(msg, false)
							task.Disabled = true
							task.reason = msg
							continue LoadLoop
						}
					}
				}
			}
			// For Go plugins, use the provided empty config struct to go ahead
			// and unmarshall Config. The GetTaskConfig call just sets a pointer
			// without unmshalling again.
			if plugin.taskType == taskGo {
				// Copy the pointer to the empty config struct / empty struct (when no config)
				// pluginHandlers[name].Config is an empty struct for unmarshalling provided
				// in RegisterPlugin.
				pt := reflect.ValueOf(pluginHandlers[task.name].Config)
				if pt.Kind() == reflect.Ptr {
					if task.Config != nil {
						// reflect magic: create a pointer to a new empty config struct for the plugin
						task.config = reflect.New(reflect.Indirect(pt).Type()).Interface()
						if err := json.Unmarshal(task.Config, task.config); err != nil {
							msg := fmt.Sprintf("Error unmarshalling plugin config json to config, disabling: %v", err)
							Log(Error, msg)
							r.debug(msg, false)
							task.Disabled = true
							task.reason = msg
							continue
						}
					} else {
						// Providing custom config not required (should it be?)
						msg := fmt.Sprintf("Plugin '%s' has custom config, but none is configured", task.name)
						Log(Warn, msg)
						r.debug(msg, false)
					}
				} else {
					if task.Config != nil {
						msg := fmt.Sprintf("Custom configuration data provided for Go plugin '%s', but no config struct was registered; disabling", task.name)
						Log(Error, msg)
						r.debug(msg, false)
						task.Disabled = true
						task.reason = msg
					} else {
						Log(Debug, fmt.Sprintf("Config interface isn't a pointer, skipping unmarshal for Go plugin '%s'", task.name))
					}
				}
			}
		}

		Log(Debug, fmt.Sprintf("Configured task '%s'", task.name))
	}
	// End of configuration loading. All invalid tasks are disabled.

	reInitPlugins := false
	currentTasks.Lock()
	currentTasks.t = tlist
	currentTasks.idMap = taskIndexByID
	currentTasks.nameMap = taskIndexByName
	currentTasks.nameSpaces = nameSpaceSet
	currentTasks.Unlock()
	// loadTaskConfig is called in initBot, before the connector has started;
	// don't init plugins in that case.
	robot.RLock()
	if robot.Connector != nil {
		reInitPlugins = true
	}
	robot.RUnlock()
	if reInitPlugins {
		initializePlugins()
	}
}
