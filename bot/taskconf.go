package bot

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"

	"github.com/ghodss/yaml"
	"github.com/lnxjedi/gopherbot/robot"
)

// loadTaskConfig() loads the configuration for all the jobs/plugins from
// /jobs/<jobname>.yaml or /plugins/<pluginname>.yaml, assigns a taskID, and
// stores the resulting array in b.tasks. Bad tasks are skipped and logged.
// Task configuration is initially loaded into temporary data structures,
// then stored in the bot package under the global bot lock.
func (c *botContext) loadTaskConfig() {
	taskIndexByID := make(map[string]int)
	taskIndexByName := make(map[string]int)
	nameSpaceSet := make(map[string]struct{})
	tlist := make([]interface{}, 0, 14)

	// Copy some data from the bot under read lock, including external plugins
	botCfg.RLock()
	defaultAllowDirect := botCfg.defaultAllowDirect
	// copy the list of default channels (for plugins only)
	pchan := botCfg.plugChannels
	jdefchan := botCfg.defaultJobChannel
	externalTasks := botCfg.externalTasks
	externalJobs := botCfg.externalJobs
	externalPlugins := botCfg.externalPlugins
	botCfg.RUnlock() // we're done with bot data 'til the end

	i := 0

	for plugname := range pluginHandlers {
		plugin := &BotPlugin{
			BotTask: &BotTask{
				name:     plugname,
				taskType: taskGo,
				taskID:   getTaskID(plugname),
			},
		}
		tlist = append(tlist, plugin)
		taskIndexByID[plugin.BotTask.taskID] = i
		taskIndexByName[plugin.BotTask.name] = i
		i++
	}

	// Initial load of plugins
	for index, script := range externalPlugins {
		if !identifierRe.MatchString(script.Name) {
			Log(robot.Error, "Plugin name: '%s', index: %d doesn't match task name regex '%s', skipping", script.Name, index+1, identifierRe.String())
			continue
		}
		if script.Name == "bot" {
			Log(robot.Error, "Illegal task name: bot - skipping")
			continue
		}
		if _, ok := taskIndexByName[script.Name]; ok {
			msg := fmt.Sprintf("External plugin index: #%d, name: '%s' duplicates name of builtIn or Go plugin, skipping", index, script.Name)
			Log(robot.Error, msg)
			continue
		}
		nameSpace := script.Name
		if len(script.NameSpace) > 0 {
			nameSpace = script.NameSpace
		}
		nameSpaceSet[nameSpace] = struct{}{}
		task := &BotTask{
			name:        script.Name,
			taskType:    taskExternal,
			taskID:      getTaskID(script.Name),
			Description: script.Description,
			Path:        script.Path,
			Parameters:  script.Parameters,
			NameSpace:   nameSpace,
		}
		if script.Disabled {
			task.Disabled = true
			task.reason = "Disabled in installed / custom gopherbot.yaml"
		}
		p := &BotPlugin{
			BotTask: task,
		}
		tlist = append(tlist, p)
		taskIndexByID[task.taskID] = i
		taskIndexByName[task.name] = i
		i++
	}

	// Initial load of jobs
	for index, script := range externalJobs {
		if !identifierRe.MatchString(script.Name) {
			Log(robot.Error, "Job name: '%s', index: %d doesn't match task name regex '%s', skipping", script.Name, index+1, identifierRe.String())
			continue
		}
		if script.Name == "bot" {
			Log(robot.Error, "Illegal task name: bot - skipping")
			continue
		}
		if _, ok := taskIndexByName[script.Name]; ok {
			msg := fmt.Sprintf("External job index: #%d, name: '%s' duplicates name of builtIn or Go plugin, skipping", index, script.Name)
			Log(robot.Error, msg)
			continue
		}
		nameSpace := script.Name
		if len(script.NameSpace) > 0 {
			nameSpace = script.NameSpace
		}
		nameSpaceSet[nameSpace] = struct{}{}
		task := &BotTask{
			name:        script.Name,
			taskType:    taskExternal,
			taskID:      getTaskID(script.Name),
			Description: script.Description,
			Path:        script.Path,
			Parameters:  script.Parameters,
			NameSpace:   nameSpace,
		}
		if script.Disabled {
			task.Disabled = true
			task.reason = "Disabled in installed / custom gopherbot.yaml"
		}
		j := &BotJob{
			BotTask: task,
		}
		tlist = append(tlist, j)
		taskIndexByID[task.taskID] = i
		taskIndexByName[task.name] = i
		i++
	}

	// Load of external tasks
	for index, script := range externalTasks {
		if !identifierRe.MatchString(script.Name) {
			Log(robot.Error, "Task name: '%s', index: %d doesn't match task name regex '%s', skipping", script.Name, index+1, identifierRe.String())
			continue
		}
		if script.Name == "bot" {
			Log(robot.Error, "Illegal task name: bot - skipping")
			continue
		}
		if _, ok := taskIndexByName[script.Name]; ok {
			Log(robot.Error, "External job index: #%d, name: '%s' duplicates name of already loaded task, skipping", index, script.Name)
			continue
		}
		nameSpace := script.Name
		if len(script.NameSpace) > 0 {
			nameSpace = script.NameSpace
		}
		nameSpaceSet[nameSpace] = struct{}{}
		task := &BotTask{
			name:        script.Name,
			taskType:    taskExternal,
			taskID:      getTaskID(script.Name),
			Description: script.Description,
			Path:        script.Path,
			Parameters:  script.Parameters,
			NameSpace:   nameSpace,
		}
		if script.Disabled {
			task.Disabled = true
			task.reason = "Disabled in installed / custom gopherbot.yaml"
		}
		tlist = append(tlist, task)
		taskIndexByID[task.taskID] = i
		taskIndexByName[task.name] = i
		i++
	}

	// Load configuration for all valid tasks. Note that this is all being loaded
	// in to non-shared data structures that will replace current configuration
	// under lock at the end.
LoadLoop:
	for _, j := range tlist {
		var plugin *BotPlugin
		var job *BotJob
		var task *BotTask
		var isPlugin bool
		switch t := j.(type) {
		case *BotPlugin:
			isPlugin = true
			plugin = t
			task = t.BotTask
		case *BotJob:
			job = t
			task = t.BotTask
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
			Log(robot.Info, "Loading configuration for plugin '%s', type %s", task.name, plugin.taskType)
		} else {
			Log(robot.Info, "Loading configuration for job '%s'", task.name)
		}

		if isPlugin {
			if plugin.taskType == taskExternal {
				// External plugins spit their default config to stdout when called with command="configure"
				cfg, err := getExtDefCfg(task)
				if err != nil {
					msg := fmt.Sprintf("Error getting default configuration for external plugin, disabling: %v", err)
					Log(robot.Error, msg)
					c.debugTask(task, msg, false)
					task.Disabled = true
					task.reason = msg
					continue
				}
				if len(*cfg) > 0 {
					c.debugTask(task, fmt.Sprintf("Loaded default config from the plugin, size: %d", len(*cfg)), false)
				} else {
					c.debugTask(task, "Unable to obtain default config from plugin, command 'configure' returned no content", false)
				}
				if err := yaml.Unmarshal(*cfg, &tcfgdefault); err != nil {
					msg := fmt.Sprintf("Error unmarshalling default configuration, disabling: %v", err)
					Log(robot.Error, "Problem unmarshalling plugin default config for '%s', disabling: %v", task.name, err)
					c.debugTask(task, msg, false)
					task.Disabled = true
					task.reason = msg
					continue
				}
			} else {
				if err := yaml.Unmarshal([]byte(pluginHandlers[task.name].DefaultConfig), &tcfgdefault); err != nil {
					msg := fmt.Sprintf("Error unmarshalling default configuration, disabling: %v", err)
					Log(robot.Error, "Problem unmarshalling plugin default config for '%s', disabling: %v", task.name, err)
					c.debugTask(task, msg, false)
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
		if err := c.getConfigFile(cpath+task.name+".yaml", task.taskID, false, tcfgload, tcfgdefault); err != nil {
			msg := fmt.Sprintf("Problem loading configuration file(s) for task '%s', disabling: %v", task.name, err)
			Log(robot.Error, msg)
			c.debugTask(task, msg, false)
			task.Disabled = true
			task.reason = msg
			continue
		}
		if disjson, ok := tcfgload["Disabled"]; ok {
			disabled := false
			if err := json.Unmarshal(disjson, &disabled); err != nil {
				msg := fmt.Sprintf("Problem unmarshalling value for 'Disabled' in plugin '%s', disabling: %v", task.name, err)
				Log(robot.Error, msg)
				c.debugTask(task, msg, false)
				task.Disabled = true
				task.reason = msg
				continue
			}
			if disabled {
				msg := fmt.Sprintf("Plugin '%s' is disabled by configuration", task.name)
				Log(robot.Info, msg)
				c.debugTask(task, msg, false)
				task.Disabled = true
				task.reason = msg
				continue
			}
		}
		// Boolean false values can be explicitly false, or default to false
		// when not specified. In some cases that matters.
		explicitAllChannels := false
		explicitAllowDirect := false

		namespace := ""

		for key, value := range tcfgload {
			var strval string
			var intval int
			var boolval bool
			var sarrval []string
			var hval []PluginHelp
			var mval []InputMatcher
			var tval []JobTrigger
			var val interface{}
			skip := false
			switch key {
			case "Elevator", "Authorizer", "AuthRequire", "NameSpace", "Channel":
				val = &strval
			case "HistoryLogs":
				val = &intval
			case "Disabled", "AllowDirect", "DirectOnly", "DenyDirect", "AllChannels", "RequireAdmin", "Protected", "AuthorizeAllCommands", "CatchAll", "MatchUnlisted", "Quiet":
				val = &boolval
			case "Channels", "ElevatedCommands", "ElevateImmediateCommands", "Users", "AuthorizedCommands", "AdminCommands":
				val = &sarrval
			case "Help":
				val = &hval
			case "CommandMatchers", "ReplyMatchers", "MessageMatchers", "Arguments":
				val = &mval
			case "Triggers":
				val = &tval
			case "Config":
				skip = true
			default:
				msg := fmt.Sprintf("Invalid configuration key for task '%s': %s - disabling", task.name, key)
				Log(robot.Error, msg)
				c.debugTask(task, msg, false)
				task.Disabled = true
				task.reason = msg
				continue LoadLoop
			}

			if !skip {
				if err := json.Unmarshal(value, val); err != nil {
					msg := fmt.Sprintf("Disabling plugin '%s' - error unmarshalling value '%s': %v", task.name, key, err)
					Log(robot.Error, msg)
					c.debugTask(task, msg, false)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				}
			}

			mismatch := false
			// Defaults
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
			case "Protected":
				task.Protected = *(val.(*bool))
			case "AdminCommands":
				if isPlugin {
					plugin.AdminCommands = *(val.(*[]string))
				} else {
					mismatch = true
				}
			case "NameSpace":
				if len(task.NameSpace) > 0 {
					msg := fmt.Sprintf("NameSpace declared in '%s.yaml' for external task, disabling", task.name)
					Log(robot.Error, msg)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				} else {
					namespace = *(val.(*string))
					if !identifierRe.MatchString(namespace) {
						Log(robot.Error, "Task '%s' has invalid NameSpace '%s'; doesn't match regex '%s', ignoring", task.name, task.NameSpace, identifierRe.String())
						namespace = ""
					}
					if namespace == "bot" {
						Log(robot.Error, "Task '%s' has illegal NameSpace 'bot', ignoring", task.name)
						namespace = ""
					}
				}
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
				if isPlugin {
					mismatch = true
				} else {
					job.HistoryLogs = *(val.(*int))
				}
			case "Authorizer":
				task.Authorizer = *(val.(*string))
			case "AuthRequire":
				task.AuthRequire = *(val.(*string))
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
			case "Arguments":
				if isPlugin {
					mismatch = true
				} else {
					job.Arguments = *(val.(*[]InputMatcher))
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
			}
			if mismatch {
				var msg string
				if isPlugin {
					msg = fmt.Sprintf("Disabling plugin '%s' - invalid configuration key: %s", task.name, key)
				} else {
					msg = fmt.Sprintf("Disabling job '%s' - invalid configuration key: %s", task.name, key)
				}
				Log(robot.Error, msg)
				c.debugTask(task, msg, false)
				task.Disabled = true
				task.reason = msg
				continue LoadLoop
			}
		}
		// End of reading configuration keys

		// Start sanity checking of configuration
		if len(task.Path) == 0 && task.taskType == taskExternal {
			msg := fmt.Sprintf("Task '%s' has zero-length path, disabling", task.name)
			Log(robot.Error, msg)
			c.debugTask(task, msg, false)
			task.Disabled = true
			task.reason = msg
		}
		// Set namespace for Go plugins
		if len(task.NameSpace) == 0 {
			task.NameSpace = task.name
			if len(namespace) > 0 {
				task.NameSpace = namespace
			}
			nameSpaceSet[task.NameSpace] = struct{}{}
		}
		if task.DirectOnly {
			if explicitAllowDirect {
				if !task.AllowDirect {
					msg := fmt.Sprintf("Task '%s' has conflicting values for AllowDirect (false) and DirectOnly (true), disabling", task.name)
					Log(robot.Error, msg)
					c.debugTask(task, msg, false)
					task.Disabled = true
					task.reason = msg
					continue
				}
			} else {
				Log(robot.Debug, "DirectOnly specified without AllowDirect; setting AllowDirect = true")
				task.AllowDirect = true
				explicitAllowDirect = true
			}
		}

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
		}

		// Considering possible default channels, is the plugin visible anywhere?
		if isPlugin {
			if len(task.Channels) > 0 {
				msg := fmt.Sprintf("Plugin '%s' will be available in channels %q", task.name, task.Channels)
				Log(robot.Info, msg)
				c.debugTask(task, msg, false)
			} else {
				if !(task.AllowDirect || task.AllChannels) {
					msg := fmt.Sprintf("Plugin '%s' not visible in any channels or by direct message, disabling", task.name)
					Log(robot.Error, msg)
					c.debugTask(task, msg, false)
					task.Disabled = true
					task.reason = msg
					continue
				} else {
					msg := fmt.Sprintf("Plugin '%s' has no channel restrictions configured; all channels: %t", task.name, task.AllChannels)
					Log(robot.Info, msg)
					c.debugTask(task, msg, false)
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
			for i := range plugin.CommandMatchers {
				command := &plugin.CommandMatchers[i]
				regex := `^\s*` + command.Regex + `\s*$`
				re, err := regexp.Compile(regex)
				if err != nil {
					msg := fmt.Sprintf("Disabling '%s', couldn't compile command regular expression '%s': %v", task.name, regex, err)
					Log(robot.Error, msg)
					c.debugTask(task, msg, false)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				} else {
					// Store the modified regex
					command.Regex = regex
					command.re = re
				}
			}
			for i := range plugin.MessageMatchers {
				// Note that full message regexes don't get the beginning and end anchors added - the individual plugin
				// will need to do this if necessary.
				message := &plugin.MessageMatchers[i]
				re, err := regexp.Compile(message.Regex)
				if err != nil {
					msg := fmt.Sprintf("Disabling '%s', couldn't compile message regular expression '%s': %v", task.name, message.Regex, err)
					Log(robot.Error, msg)
					c.debugTask(task, msg, false)
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
					c.debugTask(task, msg, false)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				}
				re, err := regexp.Compile(trigger.Regex)
				if err != nil {
					msg := fmt.Sprintf("Disabling '%s', couldn't compile trigger regular expression '%s': %v", task.name, trigger.Regex, err)
					Log(robot.Error, msg)
					c.debugTask(task, msg, false)
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
					c.debugTask(task, msg, false)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				}
				regex := `^\s*` + argument.Regex + `\s*$`
				re, err := regexp.Compile(regex)
				if err != nil {
					msg := fmt.Sprintf("Disabling '%s', couldn't compile argument regular expression '%s': %v", task.name, regex, err)
					Log(robot.Error, msg)
					c.debugTask(task, msg, false)
					task.Disabled = true
					task.reason = msg
					continue LoadLoop
				} else {
					argument.Regex = regex
					argument.re = re
				}
			}
		}
		for i := range task.ReplyMatchers {
			reply := &task.ReplyMatchers[i]
			label := reply.Label
			if stockRepliesRe.MatchString(label) {
				msg := fmt.Sprintf("Disabling '%s', invalid regex label '%s' starts with capital letter", task.name, label)
				Log(robot.Error, msg)
				c.debugTask(task, msg, false)
				task.Disabled = true
				task.reason = msg
				continue LoadLoop
			}
			re, err := regexp.Compile(`^\s*` + reply.Regex + `\s*$`)
			if err != nil {
				msg := fmt.Sprintf("Skipping %s, couldn't compile reply regular expression '%s': %v", task.name, reply.Regex, err)
				Log(robot.Error, msg)
				c.debugTask(task, msg, false)
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
							Log(robot.Error, msg)
							c.debugTask(task, msg, false)
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
							Log(robot.Error, msg)
							c.debugTask(task, msg, false)
							task.Disabled = true
							task.reason = msg
							continue
						}
					} else {
						// Providing custom config not required (should it be?)
						msg := fmt.Sprintf("Plugin '%s' has custom config, but none is configured", task.name)
						Log(robot.Warn, msg)
						c.debugTask(task, msg, false)
					}
				} else {
					if task.Config != nil {
						msg := fmt.Sprintf("Custom configuration data provided for Go plugin '%s', but no config struct was registered; disabling", task.name)
						Log(robot.Error, msg)
						c.debugTask(task, msg, false)
						task.Disabled = true
						task.reason = msg
					} else {
						Log(robot.Debug, "Config interface isn't a pointer, skipping unmarshal for Go plugin '%s'", task.name)
					}
				}
			}
		}

		Log(robot.Debug, "Configured task '%s'", task.name)
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
	botCfg.RLock()
	if botCfg.Connector != nil {
		reInitPlugins = true
	}
	botCfg.RUnlock()
	if reInitPlugins {
		initializePlugins()
	}
}
