package bot

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

const keepListeningDuration = 77 * time.Second

var spaceRe = regexp.MustCompile(" +")

// checkPluginMatchersAndRun checks either command matchers (for messages directed at
// the robot), or message matchers (for ambient commands that need not be
// directed at the robot), and calls the plugin if it matches. Note: this
// function is called under a read lock on the 'b' struct.
func (c *botContext) checkPluginMatchersAndRun(pipelineType pipelineType) (messageMatched bool) {
	r := c.makeRobot()
	// un-needed, but more clear
	messageMatched = false
	// If we're checking messages, debugging messages require that the user requested verboseness
	//verboseOnly := !checkCommands
	verboseOnly := false
	if pipelineType == plugMessage {
		verboseOnly = true
	}
	var runTask interface{}
	var matchedMatcher InputMatcher
	var cmdArgs []string
	for _, t := range c.tasks.t {
		task, plugin, _ := getTask(t)
		if plugin == nil {
			continue
		}
		if task.Disabled {
			msg := fmt.Sprintf("Skipping disabled task '%s', reason: %s", task.name, task.reason)
			Log(robot.Trace, msg)
			c.debugT(t, msg, false)
			continue
		}
		Log(robot.Trace, "Checking availability of task '%s' in channel '%s' for user '%s', active in %d channels (allchannels: %t)", task.name, c.Channel, c.User, len(task.Channels), task.AllChannels)
		ok := c.pluginAvailable(task, false, verboseOnly)
		if !ok {
			Log(robot.Trace, "Task '%s' not available for user '%s' in channel '%s', doesn't meet criteria", task.name, c.User, c.Channel)
			continue
		}
		var matchers []InputMatcher
		var ctype string
		switch pipelineType {
		case plugCommand:
			if len(plugin.CommandMatchers) == 0 {
				c.debugT(t, fmt.Sprintf("Plugin has no command matchers, skipping command check"), false)
				continue
			}
			matchers = plugin.CommandMatchers
			ctype = "command"
		case plugMessage:
			if len(plugin.MessageMatchers) == 0 {
				c.debugT(t, fmt.Sprintf("Plugin has no message matchers, skipping message check"), true)
				continue
			}
			if !c.listedUser && !plugin.MatchUnlisted && !c.isCommand {
				msg := fmt.Sprintf("ignoring unlisted user '%s' for plugin '%s' ambient messages", c.User, task.name)
				Log(robot.Trace, msg)
				c.debugT(t, msg, false)
				continue
			}
			matchers = plugin.MessageMatchers
			ctype = "message"
		}
		Log(robot.Trace, "Task '%s' is active, will check for matches", task.name)
		cmsg := spaceRe.ReplaceAllString(c.msg, " ")
		c.debugT(t, fmt.Sprintf("Checking %d %s matchers against message: '%s'", len(matchers), ctype, cmsg), verboseOnly)
		for _, matcher := range matchers {
			Log(robot.Trace, "Checking '%s' against '%s'", cmsg, matcher.Regex)
			matches := matcher.re.FindAllStringSubmatch(cmsg, -1)
			matched := false
			if matches != nil {
				c.debugT(t, fmt.Sprintf("Matched %s regex '%s', command: %s", ctype, matcher.Regex, matcher.Command), false)
				matched = true
				Log(robot.Trace, "Message '%s' matches command '%s'", cmsg, matcher.Command)
				cmdArgs = matches[0][1:]
				if len(matcher.Contexts) > 0 {
					// Resolve & store "it" with short-term memories
					ts := time.Now()
					shortTermMemories.Lock()
					for i, contextLabel := range matcher.Contexts {
						if contextLabel != "" {
							if len(cmdArgs) > i {
								ctxargs := strings.Split(contextLabel, ":")
								contextName := ctxargs[0]
								contextMatches := []string{""}
								contextMatches = append(contextMatches, ctxargs[1:]...)
								key := "context:" + contextName
								ctx := memoryContext{key, c.User, c.Channel}
								// Check if the capture group matches the empty string
								// or one of the generic values (e.g. "it")
								cMatch := false
								for _, cm := range contextMatches {
									if cmdArgs[i] == cm {
										cMatch = true
									}
								}
								if cMatch {
									// If a generic matched, try to recall from short-term memory
									s, ok := shortTermMemories.m[ctx]
									if ok {
										cmdArgs[i] = s.memory
										// TODO: it would probably be best to substitute the value
										// from "it" back in to the original message and re-check for
										// a match. Failing a match, matched should be set to false.
										s.timestamp = ts
										shortTermMemories.m[ctx] = s
									} else {
										r.Say("Sorry, I don't remember which %s we were talking about - please re-enter your command and be more specific", contextLabel)
										shortTermMemories.Unlock()
										return true
									}
								} else {
									// Didn't match generic, store the value in short-term context memory
									s := shortTermMemory{cmdArgs[i], ts}
									shortTermMemories.m[ctx] = s
								}
							} else {
								Log(robot.Error, "Plugin '%s', command '%s', has more contexts than match groups", task.name, matcher.Command)
							}
						}
					}
					shortTermMemories.Unlock()
				}
			} else {
				c.debugT(t, fmt.Sprintf("Not matched: %s", matcher.Regex), verboseOnly)
			}
			if matched {
				if messageMatched {
					prevTask, _, _ := getTask(runTask)
					Log(robot.Error, "Message '%s' matched multiple tasks: %s and %s", cmsg, prevTask.name, task.name)
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
		c.messageHeard()
		matcher := matchedMatcher
		abort := false
		if task.name == "builtin-admin" && matcher.Command == "abort" {
			abort = true
		}
		botCfg.RLock()
		if botCfg.shuttingDown && !abort {
			r.Say("Sorry, I'm shutting down and can't start any new tasks")
			botCfg.RUnlock()
			return
		}
		botCfg.RUnlock()
		// Check to see if user issued a new command when a reply was being
		// waited on
		replyMatcher := replyMatcher{c.User, c.Channel}
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
			Log(robot.Debug, "User '%s' matched a new command while the robot was waiting for a reply in channel '%s'", c.User, c.Channel)
		} else {
			replies.Unlock()
		}
		c.startPipeline(nil, runTask, pipelineType, matcher.Command, cmdArgs...)
	}
	return
}

// handleMessage checks the message against plugin commands and full-message
// matches, then dispatches it to the applicable plugin. If the robot was
// addressed directly but nothing matched, any registered CatchAll plugins are
// called. There Should Be Only One (terminal plugin called).
func (c *botContext) handleMessage() {
	privCheck("incoming message")
	r := c.makeRobot()
	defer checkPanic(r, c.msg)

	if c.directMsg {
		emit(BotDirectMessage)
		Log(robot.Trace, "Bot received a direct message from %s: %s", c.User, c.msg)
	}
	messageMatched := false
	ts := time.Now()
	lastMsgContext := memoryContext{"lastMsg", c.User, c.Channel}
	var last shortTermMemory
	var ok bool
	// See if the robot got a blank message, indicating that the last message
	// was meant for it (if it was in the keepListeningDuration)
	if c.isCommand && len(c.msg) == 0 && !c.BotUser {
		shortTermMemories.Lock()
		last, ok = shortTermMemories.m[lastMsgContext]
		shortTermMemories.Unlock()
		if ok && ts.Sub(last.timestamp) < keepListeningDuration {
			c.msg = last.memory
			messageMatched = c.checkPluginMatchersAndRun(plugCommand)
		} else {
			messageMatched = true
			r.Say("Yes?")
		}
	}
	if !messageMatched && c.isCommand {
		// See if a command matches (and runs)
		messageMatched = c.checkPluginMatchersAndRun(plugCommand)
	}
	// See if the robot was waiting on a reply
	var waiters []replyWaiter
	waitingForReply := false
	if !messageMatched {
		matcher := replyMatcher{c.User, c.Channel}
		Log(robot.Trace, "Checking replies for matcher: %q", matcher)
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
					cmsg := spaceRe.ReplaceAllString(c.msg, " ")
					matched := rep.re.MatchString(cmsg)
					Log(robot.Debug, "Found replyWaiter for user '%s' in channel '%s', checking if message '%s' matches '%s': %t", c.User, c.Channel, cmsg, rep.re.String(), matched)
					rep.replyChannel <- reply{matched, replied, cmsg}
				} else {
					Log(robot.Debug, "Sending retry to next reply waiter")
					rep.replyChannel <- reply{false, retryPrompt, ""}
				}
			}
		}
	}
	// Direct commands were checked above; if a direct command didn't match,
	// and a there wasn't a reply being waited on, then we check ambient
	// MessageMatchers.
	if !messageMatched && !c.BotUser {
		// check for ambient message matches
		messageMatched = c.checkPluginMatchersAndRun(plugMessage)
	}
	// Check for job commands
	if !messageMatched {
		messageMatched = c.checkJobMatchersAndRun()
	}
	if c.isCommand && !messageMatched && !c.BotUser { // the robot was spoken to, but nothing matched - call catchAlls
		botCfg.RLock()
		if !botCfg.shuttingDown {
			botCfg.RUnlock()
			c.messageHeard()
			Log(robot.Debug, "Unmatched command sent to robot, calling catchalls: %s", c.msg)
			emit(CatchAllsRan) // for testing, otherwise noop
			// TODO: should we allow more than 1 catchall?
			catchAllPlugins := make([]interface{}, 0, 0)
			for _, t := range c.tasks.t {
				if plugin, ok := t.(*BotPlugin); ok && plugin.CatchAll {
					catchAllPlugins = append(catchAllPlugins, t)
				}
			}
			if len(catchAllPlugins) > 1 {
				Log(robot.Error, "More than one catch all registered, none will be called")
			} else {
				// Note: if the catchall plugin has configured security, it
				// should still apply.
				if len(catchAllPlugins) != 0 {
					c.startPipeline(nil, catchAllPlugins[0], catchAll, "catchall", spaceRe.ReplaceAllString(c.msg, " "))
				} else {
					Log(robot.Debug, "Unmatched command to robot and no catchall defined")
				}
			}
		} else {
			// If the robot is shutting down, just ignore catch-all plugins
			botCfg.RUnlock()
		}
	}
	if c.BotUser {
		return
	}
	if messageMatched || c.isCommand {
		shortTermMemories.Lock()
		delete(shortTermMemories.m, lastMsgContext)
		shortTermMemories.Unlock()
	} else {
		last = shortTermMemory{c.msg, ts}
		shortTermMemories.Lock()
		shortTermMemories.m[lastMsgContext] = last
		shortTermMemories.Unlock()
	}
}
