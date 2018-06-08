package bot

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const keepListeningDuration = 77 * time.Second

// pluginAvailable checks the user and channel against the plugin's
// configuration to determine if the message should be evaluated. Used by
// both handleMessage and the help builtin. verboseOnly is set when availability
// is being checked for ambient messages or auth/elevation plugins, to indicate
// debugging verboseness.
func (r *Robot) pluginAvailable(plugin *botPlugin, helpSystem, verboseOnly bool) (available bool) {
	nvmsg := "plugin is NOT visible to user " + r.User + " in channel "
	vmsg := "plugin is visible to user " + r.User + " in channel "
	if r.directMsg {
		nvmsg += "(direct message)"
		vmsg += "(direct message)"
	} else {
		nvmsg += r.Channel
		vmsg += r.Channel
	}
	defer func(vmsg string) {
		if available {
			r.debug(plugin.taskID, vmsg, verboseOnly)
		}
	}(vmsg)
	if plugin.Disabled {
		r.debug(plugin.taskID, nvmsg+"; plugin is disabled, possibly due to configuration error", verboseOnly)
		return false
	}
	if !r.directMsg && plugin.DirectOnly && !helpSystem {
		r.debug(plugin.taskID, nvmsg+"; only available by direct message: DirectOnly is TRUE", verboseOnly)
		return false
	}
	if r.directMsg && !plugin.AllowDirect && !helpSystem {
		r.debug(plugin.taskID, nvmsg+"; not available by direct message: AllowDirect is FALSE", verboseOnly)
		return false
	}
	if plugin.RequireAdmin {
		isAdmin := false
		robot.RLock()
		for _, adminUser := range robot.adminUsers {
			if r.User == adminUser {
				isAdmin = true
				break
			}
		}
		robot.RUnlock()
		if !isAdmin {
			r.debug(plugin.taskID, nvmsg+"; RequireAdmin is TRUE and user isn't an Admin", verboseOnly)
			return false
		}
	}
	if len(plugin.Users) > 0 {
		userOk := false
		for _, allowedUser := range plugin.Users {
			match, err := filepath.Match(allowedUser, r.User)
			if match && err == nil {
				userOk = true
			}
		}
		if !userOk {
			r.debug(plugin.taskID, nvmsg+"; user is not on the list of allowed users", verboseOnly)
			return false
		}
	}
	if r.directMsg && (plugin.AllowDirect || plugin.DirectOnly) {
		return true
	}
	if len(plugin.Channels) > 0 {
		for _, pchannel := range plugin.Channels {
			if pchannel == r.Channel {
				return true
			}
		}
	} else {
		if plugin.AllChannels {
			return true
		}
	}
	if helpSystem {
		return true
	}
	r.debug(plugin.taskID, fmt.Sprintf(nvmsg+"; channel '%s' is not on the list of allowed channels: %s", r.Channel, strings.Join(plugin.Channels, ", ")), verboseOnly)
	return false
}

// checkPluginMatchersAndRun checks either command matchers (for messages directed at
// the robot), or message matchers (for ambient commands that need not be
// directed at the robot), and calls the plugin if it matches. Note: this
// function is called under a read lock on the 'b' struct.
func (bot *Robot) checkPluginMatchersAndRun(checkCommands bool) (commandMatched bool) {
	// un-needed, but more clear
	commandMatched = false
	// If we're checking messages, debugging messages require that the user requested verboseness
	verboseOnly := !checkCommands
	currentTasks.RLock()
	plugins := currentTasks.p
	currentTasks.RUnlock()
	var runPlugin *botPlugin
	var matchedMatcher InputMatcher
	var cmdArgs []string
	for _, plugin := range plugins {
		if checkCommands {
			if len(plugin.CommandMatchers) == 0 {
				bot.debug(plugin.taskID, fmt.Sprintf("Plugin has no command matchers, skipping command check"), false)
				continue
			}
		} else {
			if len(plugin.MessageMatchers) == 0 {
				bot.debug(plugin.taskID, fmt.Sprintf("Plugin has no message matchers, skipping message check"), true)
				continue
			}
		}
		Log(Trace, fmt.Sprintf("Checking availability of plugin \"%s\" in channel \"%s\" for user \"%s\", active in %d channels (allchannels: %t)", plugin.name, bot.Channel, bot.User, len(plugin.Channels), plugin.AllChannels))
		ok := bot.pluginAvailable(plugin, false, verboseOnly)
		if !ok {
			Log(Trace, fmt.Sprintf("Plugin \"%s\" not available for user \"%s\" in channel \"%s\", doesn't meet criteria", plugin.name, bot.User, bot.Channel))
			continue
		}
		Log(Trace, fmt.Sprintf("Plugin \"%s\" is active, will check for matches", plugin.name))
		var matchers []InputMatcher
		var ctype string
		if checkCommands {
			matchers = plugin.CommandMatchers
			ctype = "command"
		} else {
			matchers = plugin.MessageMatchers
			ctype = "message"
		}
		if len(matchers) > 0 {
			bot.debug(plugin.taskID, fmt.Sprintf("Checking %d %s matchers against message: \"%s\"", len(matchers), ctype, bot.msg), verboseOnly)
		}
		for _, matcher := range matchers {
			Log(Trace, fmt.Sprintf("Checking \"%s\" against \"%s\"", bot.msg, matcher.Regex))
			matches := matcher.re.FindAllStringSubmatch(bot.msg, -1)
			var matched bool
			if matches != nil {
				bot.debug(plugin.taskID, fmt.Sprintf("Matched %s regex '%s', command: %s", ctype, matcher.Regex, matcher.Command), false)
				matched = true
				Log(Trace, fmt.Sprintf("Message \"%s\" matches command \"%s\"", bot.msg, matcher.Command))
				cmdArgs = matches[0][1:]
				if len(matcher.Contexts) > 0 {
					// Resolve & store "it" with short-term memories
					ts := time.Now()
					shortTermMemories.Lock()
					for i, contextLabel := range matcher.Contexts {
						if contextLabel != "" {
							key := "context:" + contextLabel
							c := memoryContext{key, bot.User, bot.Channel}
							if len(cmdArgs) > i && (cmdArgs[i] == "it" || cmdArgs[i] == "") {
								s, ok := shortTermMemories.m[c]
								if ok {
									cmdArgs[i] = s.memory
									// TODO: it would probably be best to substitute the value
									// from "it" back in to the original message and re-check for
									// a match. Failing a match, matched should be set to false.
									s.timestamp = ts
									shortTermMemories.m[c] = s
								} else {
									bot.Say(fmt.Sprintf("Sorry, I don't remember which %s we were talking about - please re-enter your command and be more specific", contextLabel))
									shortTermMemories.Unlock()
									return true
								}
							} else {
								s := shortTermMemory{cmdArgs[i], ts}
								shortTermMemories.m[c] = s
							}
						}
					}
					shortTermMemories.Unlock()
				}
			} else {
				bot.debug(plugin.taskID, fmt.Sprintf("Not matched: %s", matcher.Regex), verboseOnly)
			}
			if matched {
				if commandMatched {
					Log(Error, fmt.Sprintf("Message \"%s\" matched multiple plugins: %s and %s", bot.msg, runPlugin.name, plugin.name))
					bot.Say("Yikes! Your command matched multiple plugins, so I'm not doing ANYTHING")
					emit(MultipleMatchesNoAction)
					return
				}
				commandMatched = true
				runPlugin = plugin
				matchedMatcher = matcher
				break
			}
		} // end of matcher checking
	} // end of plugin checking
	if commandMatched {
		bot.messageHeard()
		plugin := runPlugin
		matcher := matchedMatcher
		abort := false
		if plugin.name == "builtInadmin" && matcher.Command == "abort" {
			abort = true
		}
		robot.RLock()
		if robot.shuttingDown && !abort {
			bot.Say("Sorry, I'm shutting down and can't start any new tasks")
			robot.RUnlock()
			return
		} else if robot.paused && !abort {
			bot.Say("Sorry, I've been paused and can't start any new tasks")
			robot.RUnlock()
			return
		}
		// lazy about setting this, only if a plugin is going to run
		bot.Format = robot.defaultMessageFormat
		robot.RUnlock()
		// Check to see if user issued a new command when a reply was being
		// waited on
		replyMatcher := replyMatcher{bot.User, bot.Channel}
		replies.Lock()
		waiters, waitingForReply := replies.m[replyMatcher]
		if waitingForReply {
			delete(replies.m, replyMatcher)
			replies.Unlock()
			for i, rep := range waiters {
				if i == 0 {
					rep.replyChannel <- reply{false, replyInterrupted, ""}
				} else {
					rep.replyChannel <- reply{false, retryPrompt, ""}
				}
			}
			Log(Debug, fmt.Sprintf("User \"%s\" matched a new command while the robot was waiting for a reply in channel \"%s\"", bot.User, bot.Channel))
		} else {
			replies.Unlock()
		}
		// NOTE: if RequireAdmin is true, the user can't access the plugin at all if not an admin
		if len(plugin.AdminCommands) > 0 {
			adminRequired := false
			for _, i := range plugin.AdminCommands {
				if matcher.Command == i {
					adminRequired = true
					break
				}
			}
			if adminRequired {
				if !bot.CheckAdmin() {
					bot.Say("Sorry, that command is only available to bot administrators")
					return
				}
			}
		}
		if bot.checkAuthorization(plugins, plugin, matcher.Command, cmdArgs...) != Success {
			return
		}
		if bot.checkElevation(plugins, plugin, matcher.Command) != Success {
			return
		}
		if checkCommands {
			emit(CommandPluginRan) // for testing, otherwise noop
		} else {
			// An "ambient" message matched - not specifically directed at the robot
			emit(AmbientPluginRan) // for testing, otherwise noop
		}
		bot.debug(plugin.taskID, fmt.Sprintf("Running plugin with command '%s' and arguments: %v", matcher.Command, cmdArgs), false)
		ret := callTask(bot, plugin, true, true, matcher.Command, cmdArgs...)
		bot.debug(plugin.taskID, fmt.Sprintf("Plugin finished with return value: %s", ret), false)
	}
	return
}

// handleMessage checks the message against plugin commands and full-message
// matches, then dispatches it to the applicable plugin. If the robot was
// addressed directly but nothing matched, any registered CatchAll plugins are
// called. There Should Be Only One (terminal plugin called).
func (bot *Robot) handleMessage() {
	defer checkPanic(bot, bot.msg)

	// Get the plugins active for this message; could change while this message
	// is being handled.
	currentTasks.RLock()
	plugins := currentTasks.p
	currentTasks.RUnlock()
	if len(bot.Channel) == 0 {
		emit(BotDirectMessage)
		Log(Trace, fmt.Sprintf("Bot received a direct message from %s: %s", bot.User, bot.msg))
	}
	commandMatched := false
	var catchAllPlugins []*botPlugin
	ts := time.Now()
	lastMsgContext := memoryContext{"lastMsg", bot.User, bot.Channel}
	var last shortTermMemory
	var ok bool
	// See if the robot got a blank message, indicating that the last message
	// was meant for it (if it was in the keepListeningDuration)
	if bot.isCommand && bot.msg == "" {
		shortTermMemories.Lock()
		last, ok = shortTermMemories.m[lastMsgContext]
		shortTermMemories.Unlock()
		if ok && ts.Sub(last.timestamp) < keepListeningDuration {
			bot.msg = last.memory
			commandMatched = bot.checkPluginMatchersAndRun(true)
		} else {
			commandMatched = true
			bot.Say("Yes?")
		}
	}
	if !commandMatched && bot.isCommand {
		catchAllPlugins = make([]*botPlugin, 0, len(plugins))
		for _, plugin := range plugins {
			if plugin.CatchAll {
				catchAllPlugins = append(catchAllPlugins, plugin)
			}
		}
		// See if a command matches (and runs)
		commandMatched = bot.checkPluginMatchersAndRun(true)
	}
	// See if the robot was waiting on a reply
	var waiters []replyWaiter
	waitingForReply := false
	if !commandMatched {
		matcher := replyMatcher{bot.User, bot.Channel}
		Log(Trace, fmt.Sprintf("Checking replies for matcher: %q", matcher))
		replies.Lock()
		waiters, waitingForReply = replies.m[matcher]
		if !waitingForReply {
			replies.Unlock()
		} else {
			delete(replies.m, matcher)
			replies.Unlock()
			// if the robot was waiting on a reply, we don't want to check for
			// ambient message matches - the plugin will handle it.
			commandMatched = true
			for i, rep := range waiters {
				if i == 0 {
					matched := rep.re.MatchString(bot.msg)
					Log(Debug, fmt.Sprintf("Found replyWaiter for user \"%s\" in channel \"%s\", checking if message \"%s\" matches \"%s\": %t", bot.User, bot.Channel, bot.msg, rep.re.String(), matched))
					rep.replyChannel <- reply{matched, replied, bot.msg}
				} else {
					Log(Debug, "Sending retry to next reply waiter")
					rep.replyChannel <- reply{false, retryPrompt, ""}
				}
			}
		}
	}
	// Direct commands were checked above; if a direct command didn't match,
	// and a there wasn't a reply being waited on, then we check ambient
	// MessageMatchers.
	if !commandMatched {
		// check for ambient message matches
		commandMatched = bot.checkPluginMatchersAndRun(false)
	}
	if bot.isCommand && !commandMatched { // the robot was spoken to, but nothing matched - call catchAlls
		robot.RLock()
		if !robot.shuttingDown {
			robot.RUnlock()
			Log(Debug, fmt.Sprintf("Unmatched command sent to robot, calling catchalls: %s", bot.msg))
			emit(CatchAllsRan) // for testing, otherwise noop
			if len(catchAllPlugins) > 1 {
				bot.Log(Error, "More than one catch all registered, none will be called")
			} else {
				for _, plugin := range catchAllPlugins {
					callTask(bot, plugin, true, true, "catchall", bot.msg)
				}
			}
		} else {
			// If the robot is shutting down, just ignore catch-all plugins
			robot.RUnlock()
		}
	}
	if commandMatched || bot.isCommand {
		shortTermMemories.Lock()
		delete(shortTermMemories.m, lastMsgContext)
		shortTermMemories.Unlock()
	} else {
		last = shortTermMemory{bot.msg, ts}
		shortTermMemories.Lock()
		shortTermMemories.m[lastMsgContext] = last
		shortTermMemories.Unlock()
	}
}
