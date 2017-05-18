package bot

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

var pluginsRunning struct {
	count int
	sync.WaitGroup
	paused       bool
	shuttingDown bool
	sync.Mutex
}

// var pluginsRunning.count int
// var pluginsRunning.shuttingDown = false
// var paused = false // For Windows service pause support

// the shutdownMutex protects both the pluginsRunning.count and the pluginsRunning.shuttingDown
// flag
// var shutdownMutex sync.Mutex
// var plugRunningWaitGroup sync.WaitGroup

const keepListeningDuration = 77 * time.Second

// messageAppliesToPlugin checks the user and channel against the plugin's
// configuration to determine if the message should be evaluated. Used by
// both handleMessage and the help builtin.
func messageAppliesToPlugin(user, channel string, plugin *Plugin) bool {
	directMsg := false
	if len(channel) == 0 {
		directMsg = true
	}
	if !directMsg && plugin.DirectOnly {
		return false
	}
	if plugin.RequireAdmin {
		isAdmin := false
		robot.RLock()
		for _, adminUser := range robot.adminUsers {
			if user == adminUser {
				isAdmin = true
				break
			}
		}
		robot.RUnlock()
		if !isAdmin {
			return false
		}
	}
	if len(plugin.Users) > 0 {
		userOk := false
		for _, allowedUser := range plugin.Users {
			match, err := filepath.Match(allowedUser, user)
			if match && err == nil {
				userOk = true
			}
		}
		if !userOk {
			return false
		}
	}
	if directMsg && (plugin.AllowDirect || plugin.DirectOnly) {
		return true
	}
	if len(plugin.Channels) > 0 {
		for _, pchannel := range plugin.Channels {
			if pchannel == channel {
				return true
			}
		}
	} else {
		if plugin.AllChannels {
			return true
		}
	}
	return false
}

// checkPluginMatchers checks either command matchers (for messages directed at
// the robot), or message matchers (for ambient commands that need not be
// directed at the robot), and calls the plugin if it matches. Note: this
// function is called under a read lock on the 'b' struct.
func checkPluginMatchers(checkCommands bool, bot *Robot, messagetext string) (commandMatched bool) {
	// un-needed, but more clear
	commandMatched = false
	pluginlist.RLock()
	plugins := pluginlist.p
	pluginlist.RUnlock()
	for _, plugin := range plugins {
		Log(Trace, fmt.Sprintf("Checking message \"%s\" against plugin %s, active in %d channels (allchannels: %t)", messagetext, plugin.name, len(plugin.Channels), plugin.AllChannels))
		ok := messageAppliesToPlugin(bot.User, bot.Channel, plugin)
		if !ok {
			Log(Trace, fmt.Sprintf("Plugin %s ignoring message in channel %s, doesn't meet criteria", plugin.name, bot.Channel))
			continue
		}
		var matchers []InputMatcher
		if checkCommands {
			matchers = plugin.CommandMatchers
		} else {
			matchers = plugin.MessageMatchers
		}
		for _, matcher := range matchers {
			Log(Trace, fmt.Sprintf("Checking \"%s\" against \"%s\"", messagetext, matcher.Regex))
			matches := matcher.re.FindAllStringSubmatch(messagetext, -1)
			var matched bool
			var cmdArgs []string
			if matches != nil {
				matched = true
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
			}
			if matched {
				commandMatched = true
				privilegesOk := true
				if len(plugin.ElevatedCommands) > 0 {
					for _, i := range plugin.ElevatedCommands {
						if matcher.Command == i {
							if robot.elevator != nil {
								// elevators have their own pluginID & name, for brain access
								pbot := &Robot{
									User:    bot.User,
									Channel: bot.Channel,
									Format:  Variable,
									// NOTE: checkPluginMatchers is called under b.lock.RLock()
									pluginID: "elevator-" + robot.elevatorProvider,
								}
								privilegesOk = robot.elevator(pbot, false)
							} else {
								privilegesOk = false
								Log(Error, "Encountered elevated command and no elevation method configured")
							}
						}
					}
				}
				if len(plugin.ElevateImmediateCommands) > 0 {
					for _, i := range plugin.ElevateImmediateCommands {
						if matcher.Command == i {
							if robot.elevator != nil {
								// elevators have their own pluginID & name, for brain access
								pbot := &Robot{
									User:    bot.User,
									Channel: bot.Channel,
									Format:  Variable,
									// NOTE: checkPluginMatchers is called under b.lock.RLock()
									pluginID: "elevator-" + robot.elevatorProvider,
								}
								privilegesOk = robot.elevator(pbot, true)
							} else {
								privilegesOk = false
								Log(Error, "Encountered elevated command and no elevation method configured")
							}
						}
					}
				}
				if privilegesOk {
					abort := false
					if plugin.name == "builtInadmin" && matcher.Command == "abort" {
						abort = true
					}
					pluginsRunning.Lock()
					if pluginsRunning.shuttingDown && !abort {
						bot.Say("Sorry, I'm shutting down and can't start any new tasks")
						pluginsRunning.Unlock()
					} else if pluginsRunning.paused && !abort {
						bot.Say("Sorry, I've been paused and can't start any new tasks")
						pluginsRunning.Unlock()
					} else {
						pluginsRunning.Unlock()
						pluginsRunning.Add(1)
						go callPlugin(bot, plugin, true, matcher.Command, cmdArgs...)
					}
				} else {
					Log(Error, fmt.Sprintf("Elevation failed for command \"%s\", plugin %s", matcher.Command, plugin.name))
					bot.Say(fmt.Sprintf("Sorry, the \"%s\" command requires elevated privileges", matcher.Command))
				}
			}
		}
	}
	return commandMatched
}

// handleMessage checks the message against plugin commands and full-message matches,
// then dispatches it to all applicable handlers in a separate go routine. If the robot
// was addressed directly but nothing matched, any registered CatchAll plugins are called.
// There Should Be Only One (catchall, in theory (?))
func handleMessage(isCommand bool, channel, user, messagetext string) {
	bot := &Robot{
		User:    user,
		Channel: channel,
		Format:  Variable,
	}
	defer checkPanic(bot, messagetext)
	pluginlist.RLock()
	plugins := pluginlist.p
	pluginlist.RUnlock()
	if len(channel) == 0 {
		Log(Trace, fmt.Sprintf("Bot received a direct message from %s: %s", user, messagetext))
	}
	commandMatched := false
	var catchAllPlugins []*Plugin
	ts := time.Now()
	lastMsgContext := memoryContext{"lastMsg", user, channel}
	var last shortTermMemory
	var ok bool
	// See if the robot got a blank message, indicating that the last message
	// was meant for it (if it was in the keepListeningDuration)
	if isCommand && messagetext == "" {
		shortTermMemories.Lock()
		last, ok = shortTermMemories.m[lastMsgContext]
		shortTermMemories.Unlock()
		if ok && ts.Sub(last.timestamp) < keepListeningDuration {
			messagetext = last.memory
			commandMatched = checkPluginMatchers(true, bot, messagetext)
		} else {
			commandMatched = true
			bot.Say("Yes?")
		}
	}
	if !commandMatched && isCommand {
		catchAllPlugins = make([]*Plugin, 0, len(plugins))
		for _, plugin := range plugins {
			if plugin.CatchAll {
				catchAllPlugins = append(catchAllPlugins, plugin)
			}
		}
		// See if a command matches (and runs)
		commandMatched = checkPluginMatchers(true, bot, messagetext)
	}
	// See if the robot was waiting on a reply
	matcher := replyMatcher{user, channel}
	Log(Trace, fmt.Sprintf("Checking replies for matcher: %q", matcher))
	replies.Lock()
	rep, waitingForReply := replies.m[matcher]
	if !waitingForReply {
		replies.Unlock()
		Log(Trace, "No matching replyWaiter")
	} else {
		delete(replies.m, matcher)
		replies.Unlock()
		if commandMatched {
			rep.replyChannel <- reply{false, true, ""}
			Log(Debug, fmt.Sprintf("User \"%s\" issued a new command while the robot was waiting for a reply in channel \"%s\"", user, channel))
		} else {
			// if the robot was waiting on a reply, we don't want to check for
			// ambient message matches - the plugin will handle it.
			commandMatched = true
			matched := false
			if rep.re.MatchString(messagetext) {
				matched = true
			}
			Log(Debug, fmt.Sprintf("Found replyWaiter for user \"%s\" in channel \"%s\", checking if message \"%s\" matches \"%s\": %t", user, channel, messagetext, rep.re.String(), matched))
			rep.replyChannel <- reply{matched, false, messagetext}
		}
	}
	// Direct commands were checked above; if a direct command didn't match,
	// and a there wasn't a reply being waited on, then we check ambient
	// MessageMatchers if it wasn't a direct command. Note that ambient
	// commands never match in a DM.
	if !commandMatched && !waitingForReply && !isCommand {
		// check for ambient message matches
		commandMatched = checkPluginMatchers(false, bot, messagetext)
	}
	if isCommand && !commandMatched { // the robot was spoken too, but nothing matched - call catchAlls
		pluginsRunning.Lock()
		if !pluginsRunning.shuttingDown {
			pluginsRunning.Unlock()
			Log(Debug, fmt.Sprintf("Unmatched command sent to robot, calling catchalls: %s", messagetext))
			for _, plugin := range catchAllPlugins {
				pluginsRunning.Add(1)
				go callPlugin(bot, plugin, true, "catchall", messagetext)
			}
		} else {
			// If the robot is shutting down, just ignore catch-all plugins
			pluginsRunning.Unlock()
		}
	}
	if commandMatched || isCommand {
		shortTermMemories.Lock()
		delete(shortTermMemories.m, lastMsgContext)
		shortTermMemories.Unlock()
	} else {
		last = shortTermMemory{messagetext, ts}
		shortTermMemories.Lock()
		shortTermMemories.m[lastMsgContext] = last
		shortTermMemories.Unlock()
	}
}
