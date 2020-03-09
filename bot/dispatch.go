package bot

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lnxjedi/robot"
)

const keepListeningDuration = 77 * time.Second

var spaceRe = regexp.MustCompile(" +")

// checkPluginMatchersAndRun checks either command matchers (for messages directed at
// the robot), or message matchers (for ambient commands that need not be
// directed at the robot), and calls the plugin if it matches. Note: this
// function is called under a read lock on the 'b' struct.
func (w *worker) checkPluginMatchersAndRun(pipelineType pipelineType) (messageMatched bool) {
	// un-needed, but more clear
	messageMatched = false
	// If we're checking messages, debugging messages require that the admin requested verboseness
	verboseOnly := false
	if pipelineType == plugMessage {
		verboseOnly = true
	}
	var runTask interface{}
	var matchedMatcher InputMatcher
	var cmdArgs []string
	// Note: skip the first task, dummy used for namespaces
	for _, t := range w.tasks.t[1:] {
		task, plugin, _ := getTask(t)
		if plugin == nil {
			continue
		}
		if task.Disabled {
			msg := fmt.Sprintf("Skipping disabled task '%s', reason: %s", task.name, task.reason)
			Log(robot.Trace, msg)
			debugT(t, msg, false)
			continue
		}
		Log(robot.Trace, "Checking availability of task '%s' in channel '%s' for user '%s', active in %d channels (allchannels: %t)", task.name, w.Channel, w.User, len(task.Channels), task.AllChannels)
		ok := w.pluginAvailable(task, false, verboseOnly)
		if !ok {
			Log(robot.Trace, "Task '%s' not available for user '%s' in channel '%s', doesn't meet criteria", task.name, w.User, w.Channel)
			continue
		}
		var matchers []InputMatcher
		var ctype string
		switch pipelineType {
		case plugCommand:
			if len(plugin.CommandMatchers) == 0 {
				debugT(t, fmt.Sprintf("Plugin has no command matchers, skipping command check"), false)
				continue
			}
			matchers = plugin.CommandMatchers
			ctype = "command"
		case plugMessage:
			if len(plugin.MessageMatchers) == 0 {
				debugT(t, fmt.Sprintf("Plugin has no message matchers, skipping message check"), true)
				continue
			}
			if !w.listedUser && !plugin.MatchUnlisted && !w.isCommand {
				msg := fmt.Sprintf("ignoring unlisted user '%s' for plugin '%s' ambient messages", w.User, task.name)
				Log(robot.Trace, msg)
				debugT(t, msg, false)
				continue
			}
			matchers = plugin.MessageMatchers
			ctype = "message"
		}
		Log(robot.Trace, "Task '%s' is active, will check for matches", task.name)
		cmsg := spaceRe.ReplaceAllString(w.msg, " ")
		debugT(t, fmt.Sprintf("Checking %d %s matchers against message: '%s'", len(matchers), ctype, cmsg), verboseOnly)
		for _, matcher := range matchers {
			Log(robot.Trace, "Checking '%s' against '%s'", cmsg, matcher.Regex)
			matches := matcher.re.FindAllStringSubmatch(cmsg, -1)
			matched := false
			if matches != nil {
				debugT(t, fmt.Sprintf("Matched %s regex '%s', command: %s", ctype, matcher.Regex, matcher.Command), false)
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
								ctx := memoryContext{key, w.User, w.Channel}
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
										w.Say("Sorry, I don't remember which %s we were talking about - please re-enter your command and be more specific", contextLabel)
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
				debugT(t, fmt.Sprintf("Not matched: %s", matcher.Regex), verboseOnly)
			}
			if matched {
				if messageMatched {
					prevTask, _, _ := getTask(runTask)
					Log(robot.Error, "Message '%s' matched multiple tasks: %s and %s", cmsg, prevTask.name, task.name)
					w.Say("Yikes! Your command matched multiple plugins, so I'm not doing ANYTHING")
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
		w.messageHeard()
		matcher := matchedMatcher
		abort := false
		if task.name == "builtin-admin" && matcher.Command == "abort" {
			abort = true
		}
		state.RLock()
		if state.shuttingDown && !abort {
			w.Say("Sorry, I'm shutting down and can't start any new tasks")
			state.RUnlock()
			return
		}
		state.RUnlock()
		w.startPipeline(nil, runTask, pipelineType, matcher.Command, cmdArgs...)
	}
	return
}

// handleMessage checks the message against plugin commands and full-message
// matches, then dispatches it to the applicable plugin. If the robot was
// addressed directly but nothing matched, any registered CatchAll plugins are
// called. There Should Be Only One (terminal plugin called).
func (w *worker) handleMessage() {
	defer checkPanic(w, w.msg)

	if w.directMsg {
		emit(BotDirectMessage)
		Log(robot.Trace, "Bot received a direct message from %s: %s", w.User, w.msg)
	}
	messageMatched := false
	ts := time.Now()
	lastMsgContext := memoryContext{"lastMsg", w.User, w.Channel}
	var last shortTermMemory
	var ok bool
	// First, see if the robot was waiting on a reply; replies from
	// user take precedence over everything else.
	var waiters []replyWaiter
	waitingForReply := false
	matcher := replyMatcher{w.User, w.Channel}
	Log(robot.Trace, "Checking replies for matcher: %q", matcher)
	replies.Lock()
	waiters, waitingForReply = replies.m[matcher]
	if !waitingForReply {
		replies.Unlock()
	} else {
		delete(replies.m, matcher)
		replies.Unlock()
		// if the robot was waiting on a reply from this user, it always
		// counts as a matched message.
		messageMatched = true
		for i, rep := range waiters {
			if i == 0 {
				cmsg := spaceRe.ReplaceAllString(w.msg, " ")
				matched := rep.re.MatchString(cmsg)
				Log(robot.Debug, "Found replyWaiter for user '%s' in channel '%s', checking if message '%s' matches '%s': %t", w.User, w.Channel, cmsg, rep.re.String(), matched)
				rep.replyChannel <- reply{matched, replied, cmsg}
			} else {
				Log(robot.Debug, "Sending retry to next reply waiter")
				rep.replyChannel <- reply{false, retryPrompt, ""}
			}
		}
	}
	// See if the robot got a blank message, indicating that the last message
	// was meant for it (if it was in the keepListeningDuration)
	if !messageMatched && w.isCommand && len(w.msg) == 0 && !w.BotUser {
		shortTermMemories.Lock()
		last, ok = shortTermMemories.m[lastMsgContext]
		shortTermMemories.Unlock()
		if ok && ts.Sub(last.timestamp) < keepListeningDuration {
			w.msg = last.memory
			messageMatched = w.checkPluginMatchersAndRun(plugCommand)
		} else {
			messageMatched = true
			w.Say("Yes?")
		}
	}
	if !messageMatched && w.isCommand {
		// See if a command matches (and runs)
		messageMatched = w.checkPluginMatchersAndRun(plugCommand)
	}
	// Direct commands were checked above; if a direct command didn't match,
	// and a there wasn't a reply being waited on, then we check ambient
	// MessageMatchers.
	if !messageMatched && !w.BotUser {
		// check for ambient message matches
		messageMatched = w.checkPluginMatchersAndRun(plugMessage)
	}
	// Check for job commands
	if !messageMatched {
		messageMatched = w.checkJobMatchersAndRun()
	}
	if w.isCommand && !messageMatched && !w.BotUser { // the robot was spoken to, but nothing matched - call catchAlls
		state.RLock()
		if !state.shuttingDown {
			state.RUnlock()
			w.messageHeard()
			Log(robot.Debug, "Unmatched command sent to robot, calling catchalls: %s", w.msg)
			emit(CatchAllsRan) // for testing, otherwise noop
			// TODO: should we allow more than 1 catchall?
			catchAllPlugins := make([]interface{}, 0, 0)
			for _, t := range w.tasks.t[1:] {
				if plugin, ok := t.(*Plugin); ok && plugin.CatchAll {
					catchAllPlugins = append(catchAllPlugins, t)
				}
			}
			if len(catchAllPlugins) > 1 {
				Log(robot.Error, "More than one catch all registered, none will be called")
			} else {
				// Note: if the catchall plugin has configured security, it
				// should still apply.
				if len(catchAllPlugins) != 0 {
					w.startPipeline(nil, catchAllPlugins[0], catchAll, "catchall", spaceRe.ReplaceAllString(w.msg, " "))
				} else {
					Log(robot.Debug, "Unmatched command to robot and no catchall defined")
				}
			}
		} else {
			// If the robot is shutting down, just ignore catch-all plugins
			state.RUnlock()
		}
	}
	if w.BotUser {
		return
	}
	if messageMatched || w.isCommand {
		shortTermMemories.Lock()
		delete(shortTermMemories.m, lastMsgContext)
		shortTermMemories.Unlock()
	} else {
		last = shortTermMemory{w.msg, ts}
		shortTermMemories.Lock()
		shortTermMemories.m[lastMsgContext] = last
		shortTermMemories.Unlock()
	}
}
