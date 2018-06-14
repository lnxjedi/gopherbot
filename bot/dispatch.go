package bot

import (
	"fmt"
	"time"
)

const keepListeningDuration = 77 * time.Second

// checkTaskMatchersAndRun checks either command matchers (for messages directed at
// the robot), or message matchers (for ambient commands that need not be
// directed at the robot), and calls the plugin if it matches. Note: this
// function is called under a read lock on the 'b' struct.
func (bot *botContext) checkTaskMatchersAndRun(matcherType matcherType) (messageMatched bool) {
	r := bot.makeRobot()
	// un-needed, but more clear
	messageMatched = false
	// If we're checking messages, debugging messages require that the user requested verboseness
	//verboseOnly := !checkCommands
	verboseOnly := false
	if matcherType == plugMessages || matcherType == jobTriggers {
		verboseOnly = true
	}
	var runTask interface{}
	var matchedMatcher InputMatcher
	var cmdArgs []string
	for _, t := range bot.tasks.t {
		task, plugin, _ := getTask(t)
		Log(Trace, fmt.Sprintf("Checking availability of task '%s' in channel '%s' for user '%s', active in %d channels (allchannels: %t)", task.name, bot.Channel, bot.User, len(task.Channels), task.AllChannels))
		ok := bot.taskAvailable(task, false, verboseOnly)
		if !ok {
			Log(Trace, fmt.Sprintf("Task '%s' not available for user '%s' in channel '%s', doesn't meet criteria", task.name, bot.User, bot.Channel))
			continue
		}
		var matchers []InputMatcher
		var ctype string
		switch matcherType {
		case plugCommands:
			if plugin == nil {
				continue
			}
			if len(plugin.CommandMatchers) == 0 {
				bot.debug(fmt.Sprintf("Plugin has no command matchers, skipping command check"), false)
				continue
			}
			matchers = plugin.CommandMatchers
			ctype = "command"
		case plugMessages:
			if plugin == nil {
				continue
			}
			if len(plugin.MessageMatchers) == 0 {
				bot.debug(fmt.Sprintf("Plugin has no message matchers, skipping message check"), true)
				continue
			}
			matchers = plugin.MessageMatchers
			ctype = "message"
		}
		Log(Trace, fmt.Sprintf("Task '%s' is active, will check for matches", task.name))
		bot.debug(fmt.Sprintf("Checking %d %s matchers against message: '%s'", len(matchers), ctype, bot.msg), verboseOnly)
		for _, matcher := range matchers {
			Log(Trace, fmt.Sprintf("Checking '%s' against '%s'", bot.msg, matcher.Regex))
			matches := matcher.re.FindAllStringSubmatch(bot.msg, -1)
			matched := false
			if matches != nil {
				bot.debug(fmt.Sprintf("Matched %s regex '%s', command: %s", ctype, matcher.Regex, matcher.Command), false)
				matched = true
				Log(Trace, fmt.Sprintf("Message '%s' matches command '%s'", bot.msg, matcher.Command))
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
									bot.makeRobot().Say(fmt.Sprintf("Sorry, I don't remember which %s we were talking about - please re-enter your command and be more specific", contextLabel))
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
				bot.debug(fmt.Sprintf("Not matched: %s", matcher.Regex), verboseOnly)
			}
			if matched {
				if messageMatched {
					prevTask, _, _ := getTask(runTask)
					Log(Error, fmt.Sprintf("Message '%s' matched multiple tasks: %s and %s", bot.msg, prevTask.name, task.name))
					r.Say("Yikes! Your command matched multiple plugins, so I'm not doing ANYTHING")
					emit(MultipleMatchesNoAction)
					return
				}
				messageMatched = true
				runTask = t
				matchedMatcher = matcher
				break
			}
		} // end of matcher checking
	} // end of plugin checking
	if messageMatched {
		task, _, _ := getTask(runTask)
		r.messageHeard()
		matcher := matchedMatcher
		abort := false
		if task.name == "builtInadmin" && matcher.Command == "abort" {
			abort = true
		}
		robot.RLock()
		if robot.shuttingDown && !abort {
			r.Say("Sorry, I'm shutting down and can't start any new tasks")
			robot.RUnlock()
			return
		} else if robot.paused && !abort {
			r.Say("Sorry, I've been paused and can't start any new tasks")
			robot.RUnlock()
			return
		}
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
			Log(Debug, fmt.Sprintf("User '%s' matched a new command while the robot was waiting for a reply in channel '%s'", bot.User, bot.Channel))
		} else {
			replies.Unlock()
		}
		matcher.matcherType = matcherType
		bot.runPipeline(runTask, true, &matcher, cmdArgs...)
	}
	return
}

// handleMessage checks the message against plugin commands and full-message
// matches, then dispatches it to the applicable plugin. If the robot was
// addressed directly but nothing matched, any registered CatchAll plugins are
// called. There Should Be Only One (terminal plugin called).
func (bot *botContext) handleMessage() {
	r := bot.makeRobot()
	defer checkPanic(r, bot.msg)

	if len(bot.Channel) == 0 {
		emit(BotDirectMessage)
		Log(Trace, fmt.Sprintf("Bot received a direct message from %s: %s", bot.User, bot.msg))
	}
	messageMatched := false
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
			messageMatched = bot.checkTaskMatchersAndRun(plugCommands)
		} else {
			messageMatched = true
			r.Say("Yes?")
		}
	}
	if !messageMatched && bot.isCommand {
		// See if a command matches (and runs)
		messageMatched = bot.checkTaskMatchersAndRun(plugCommands)
	}
	// See if the robot was waiting on a reply
	var waiters []replyWaiter
	waitingForReply := false
	if !messageMatched {
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
			messageMatched = true
			for i, rep := range waiters {
				if i == 0 {
					matched := rep.re.MatchString(bot.msg)
					Log(Debug, fmt.Sprintf("Found replyWaiter for user '%s' in channel '%s', checking if message '%s' matches '%s': %t", bot.User, bot.Channel, bot.msg, rep.re.String(), matched))
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
	if !messageMatched {
		// check for ambient message matches
		messageMatched = bot.checkTaskMatchersAndRun(plugMessages)
	}
	if bot.isCommand && !messageMatched { // the robot was spoken to, but nothing matched - call catchAlls
		robot.RLock()
		if !robot.shuttingDown {
			robot.RUnlock()
			r.messageHeard()
			Log(Debug, fmt.Sprintf("Unmatched command sent to robot, calling catchalls: %s", bot.msg))
			emit(CatchAllsRan) // for testing, otherwise noop
			// TODO: should we allow more than 1 catchall?
			catchAllPlugins := make([]interface{}, 0, 0)
			for _, t := range bot.tasks.t {
				if plugin, ok := t.(*botPlugin); ok && plugin.CatchAll {
					catchAllPlugins = append(catchAllPlugins, t)
				}
			}
			if len(catchAllPlugins) > 1 {
				Log(Error, "More than one catch all registered, none will be called")
			} else {
				// Note: if the catchall plugin has configured security, it
				// should still apply.
				matcher := InputMatcher{
					Command:     "catchall",
					matcherType: catchAll,
				}
				bot.runPipeline(catchAllPlugins[0], true, &matcher, bot.msg)
			}
		} else {
			// If the robot is shutting down, just ignore catch-all plugins
			robot.RUnlock()
		}
	}
	if messageMatched || bot.isCommand {
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
