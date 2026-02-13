package bot

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

const keepListeningDuration = 77 * time.Second

var spaceRe = regexp.MustCompile(`\s+`)

const lastMsgKey = "lastMsg"

// checkPluginMatchersAndRun checks either command matchers (for messages directed at
// the robot), or message matchers (for ambient commands that need not be
// directed at the robot), and calls the plugin if it matches. Note: this
// function is called under a read lock on the 'b' struct.
func (w *worker) checkPluginMatchersAndRun(pipelineType pipelineType) (messageMatched bool) {
	// un-needed, but more clear
	messageMatched = false
	matchMsg := w.msg
	// If we're checking messages, debugging messages require that the admin requested verboseness
	verboseOnly := false
	if pipelineType == plugMessage {
		verboseOnly = true
		matchMsg = w.fmsg
	}
	matchChannelOnly := false
	if len(w.Channel) > 0 && !w.Incoming.ThreadedMessage {
		matchChannelOnly = true
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
			continue
		}
		Log(robot.Trace, "Checking availability of task '%s' in channel '%s' for user '%s', active in %d channels (allchannels: %t)", task.name, w.Channel, w.User, len(task.Channels), task.AllChannels)
		ok, _ := w.pluginAvailable(task, false, verboseOnly)
		if !ok {
			Log(robot.Trace, "Task '%s' not available for user '%s' in channel '%s', doesn't meet criteria", task.name, w.User, w.Channel)
			continue
		}
		var matchers []InputMatcher
		switch pipelineType {
		case plugCommand:
			if len(plugin.CommandMatchers) == 0 {
				continue
			}
			matchers = plugin.CommandMatchers
		case plugMessage:
			if w.isCommand && !plugin.AmbientMatchCommand {
				continue
			}
			if len(plugin.MessageMatchers) == 0 {
				continue
			}
			if !w.listedUser && !plugin.MatchUnlisted && !w.isCommand {
				msg := fmt.Sprintf("ignoring unlisted user '%s' for plugin '%s' ambient messages", w.User, task.name)
				Log(robot.Debug, msg)
				continue
			}
			matchers = plugin.MessageMatchers
		}
		Log(robot.Trace, "Task '%s' is active, will check for matches", task.name)
		cmsg := spaceRe.ReplaceAllString(matchMsg, " ")
		for _, matcher := range matchers {
			if matcher.ChannelOnly && !matchChannelOnly {
				Log(robot.Trace, "Skipping '%s', requested ChannelOnly matching", matcher.Regex)
				continue
			}
			Log(robot.Trace, "Checking '%s' against '%s'", cmsg, matcher.Regex)
			matches := matcher.re.FindStringSubmatch(matchMsg)
			if matches != nil {
				cmsg = w.msg
			} else {
				matches = matcher.re.FindStringSubmatch(cmsg)
			}
			matched := false
			if matches != nil {
				matched = true
				Log(robot.Trace, "Message '%s' matches command '%s'", cmsg, matcher.Command)
				cmdArgs = matches[1:]
				if len(matcher.Contexts) > 0 {
					// Resolve & store "it" with ephemeral memories
					ts := time.Now()
					modified := false
					ephemeralMemories.Lock()
					for i, contextLabel := range matcher.Contexts {
						if contextLabel != "" {
							if len(cmdArgs) > i {
								ctxargs := strings.Split(contextLabel, ":")
								contextName := ctxargs[0]
								contextMatches := []string{""}
								contextMatches = append(contextMatches, ctxargs[1:]...)
								key := "context:" + contextName
								ctx := w.makeMemoryContext(key)
								// Check if the capture group matches the empty string
								// or one of the generic values (e.g. "it")
								cMatch := false
								for _, cm := range contextMatches {
									if cmdArgs[i] == cm {
										cMatch = true
									}
								}
								if cMatch {
									// If a generic matched, try to recall from ephemeral memory
									s, ok := ephemeralMemories.m[ctx]
									if ok {
										cmdArgs[i] = s.Memory
										// TODO: it would probably be best to substitute the value
										// from "it" back in to the original message and re-check for
										// a match. Failing a match, matched should be set to false.
										s.Timestamp = ts
										ephemeralMemories.m[ctx] = s
										modified = true
									} else {
										w.Say("Sorry, I don't remember which %s we were talking about - please re-enter your command and be more specific", contextLabel)
										ephemeralMemories.Unlock()
										return true
									}
								} else {
									// Didn't match generic, store the value in ephemeral context memory
									s := ephemeralMemory{cmdArgs[i], ts}
									ephemeralMemories.m[ctx] = s
									modified = true
								}
							} else {
								Log(robot.Error, "Plugin '%s', command '%s', has more contexts than match groups", task.name, matcher.Command)
							}
						}
					}
					if modified {
						ephemeralMemories.dirty = true
					}
					ephemeralMemories.Unlock()
				}
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
		allow := false
		if task.name == "builtin-admin" {
			switch matcher.Command {
			case "ps", "kill", "abort":
				allow = true
			}
		}
		state.RLock()
		if state.shuttingDown && !allow {
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

	if w.Incoming.DirectMessage {
		emit(BotDirectMessage)
		Log(robot.Trace, "Bot received a direct message from %s: %s", w.User, w.msg)
	}
	messageMatched := false
	ts := time.Now()
	lastMsgContext := w.makeMemoryContext(lastMsgKey)
	var last ephemeralMemory
	var ok bool
	// First, see if the robot was waiting on a reply; replies from
	// user take precedence over everything else.
	var waiters []replyWaiter
	waitingForReply := false
	threadID := ""
	if w.Incoming.ThreadedMessage {
		threadID = w.Incoming.ThreadID
	}
	incomingProtocol := protocolFromIncoming(w.Incoming, w.Protocol)
	matcher := replyMatcher{
		protocol: incomingProtocol,
		user:     w.User,
		channel:  w.Channel,
		thread:   threadID,
	}
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
				cmsg := spaceRe.ReplaceAllString(w.fmsg, " ")
				matched := rep.re.MatchString(w.fmsg)
				if matched {
					cmsg = w.fmsg
				} else {
					matched = rep.re.MatchString(cmsg)
				}
				Log(robot.Debug, "Found replyWaiter for user '%s' in channel '%s'/thread '%s', checking if message '%s' matches '%s': %t", w.User, w.Channel, w.Incoming.ThreadID, cmsg, rep.re.String(), matched)
				rep.replyChannel <- reply{matched, replied, cmsg}
			} else {
				Log(robot.Debug, "Sending retry to next reply waiter")
				rep.replyChannel <- reply{false, retryPrompt, ""}
			}
		}
	}
	// See if the robot got a blank message, indicating that the last message
	// was meant for it (if it was in the keepListeningDuration); also handle "robot?"
	// This happens when the bareRegex matches.
	if !messageMatched && w.isCommand && !w.Incoming.SelfMessage && len(w.msg) == 0 && !w.BotUser {
		ephemeralMemories.Lock()
		last, ok = ephemeralMemories.m[lastMsgContext]
		ephemeralMemories.Unlock()
		Log(robot.Debug, "Barename/blank message to robot received ('%s'), checking last message: '%s'", w.fmsg, last.Memory)
		// Allow individual plugins to handle a lone "?"
		// Feature added for - you guessed it - the AI plugin
		if strings.HasSuffix(w.fmsg, "?") {
			w.msg = "?"
			messageMatched = w.checkPluginMatchersAndRun(plugCommand)
		}
		if !messageMatched {
			if ok && ts.Sub(last.Timestamp) < keepListeningDuration {
				w.msg = last.Memory
				messageMatched = w.checkPluginMatchersAndRun(plugCommand)
			} else {
				messageMatched = true
				w.Say("Yes?")
			}
		}
	}
	// NOTE: Another bot can send a plugCommand to the bot
	if !messageMatched && w.isCommand && !w.Incoming.SelfMessage {
		// See if a command matches (and runs)
		messageMatched = w.checkPluginMatchersAndRun(plugCommand)
	}
	// Direct commands were checked above; if a direct command didn't match,
	// and a there wasn't a reply being waited on, then we check ambient
	// MessageMatchers.
	if !messageMatched && !w.Incoming.SelfMessage && !w.BotUser {
		// check for ambient message matches
		messageMatched = w.checkPluginMatchersAndRun(plugMessage)
	}
	// Check for job commands
	if !messageMatched {
		messageMatched = w.checkJobMatchersAndRun()
	}
	catchAllMatched := false
	if w.isCommand && !messageMatched && !w.Incoming.SelfMessage && !w.BotUser { // the robot was spoken to, but nothing matched - call catchAlls
		state.RLock()
		if !state.shuttingDown {
			state.RUnlock()
			w.messageHeard()
			Log(robot.Debug, "Unmatched command sent to robot, calling catchalls: %s", w.msg)
			emit(CatchAllsRan) // for testing, otherwise noop
			var specificCatchAll, fallbackCatchAll interface{}
			var multipleCatchallMatched, multipleFallbackMatched bool
			for _, t := range w.tasks.t[1:] {
				task, plugin, _ := getTask(t)
				if plugin == nil || !plugin.CatchAll {
					Log(robot.Trace, "Checking plugin %s for catch-all (false)", task.name)
					continue
				}
				available, specific := w.pluginAvailable(task, false, false)
				if !available {
					continue
				}
				if specific {
					Log(robot.Trace, "Checking plugin %s for catch-all (true, specific)", task.name)
					if specificCatchAll == nil {
						specificCatchAll = t
					} else {
						multipleCatchallMatched = true
						break
					}
				} else {
					Log(robot.Trace, "Checking plugin %s for catch-all (true, non-specific)", task.name)
					if fallbackCatchAll == nil {
						fallbackCatchAll = t
					} else {
						multipleFallbackMatched = true
					}
				}
			}
			if multipleCatchallMatched {
				Log(robot.Error, "More than one specific catch-all matched, none will be called")
			} else {
				if specificCatchAll != nil {
					task, _, _ := getTask(specificCatchAll)
					Log(robot.Debug, "Unmatched command, calling specific catchall '%s' in channel '%s'", task.name, w.Channel)
					catchAllMatched = true
					w.startPipeline(nil, specificCatchAll, catchAll, "catchall", w.fmsg)
				} else if fallbackCatchAll != nil {
					if multipleFallbackMatched {
						Log(robot.Error, "More than one fallback catch-all matched, none will be called")
					} else {
						task, _, _ := getTask(fallbackCatchAll)
						Log(robot.Debug, "Unmatched command, calling fallback catchall '%s' in channel '%s'", task.name, w.Channel)
						catchAllMatched = true
						w.startPipeline(nil, fallbackCatchAll, catchAll, "catchall", w.fmsg)
					}
				} else {
					Log(robot.Debug, "Unmatched command to robot and no catchall defined")
				}
			}
		} else {
			// If the robot is shutting down, just ignore catch-all plugins
			state.RUnlock()
		}
	}
	// Last of all, check for thread subscriptions
	if !messageMatched && !w.Incoming.SelfMessage && (w.isCommand && !catchAllMatched || !w.isCommand) {
		subscriptionSpec := subscriptionMatcher{w.Channel, w.Incoming.ThreadID}
		subscriptions.Lock()
		if subscription, ok := subscriptions.m[subscriptionSpec]; ok {
			subscription.Timestamp = time.Now()
			subscriptions.Unlock()
			t := w.tasks.getTaskByName(subscription.Plugin)
			if w.Incoming.UserID != w.cfg.botinfo.UserID {
				Log(robot.Debug, "Unmatched message being routed to thread subscriber '%s' in thread '%s', channel '%s'", subscription.Plugin, w.Incoming.ThreadID, w.Channel)
				w.startPipeline(nil, t, plugThreadSubscription, "subscribed", w.fmsg)
			} else {
				Log(robot.Debug, "Ignoring message from the robot after subscription matched for thread subscriber '%s' in thread '%s', channel '%s'")
			}
		} else {
			subscriptions.Unlock()
		}
	}
	if w.BotUser {
		return
	}
	if messageMatched || w.isCommand {
		ephemeralMemories.Lock()
		delete(ephemeralMemories.m, lastMsgContext)
		ephemeralMemories.Unlock()
	} else {
		last = ephemeralMemory{w.msg, ts}
		ephemeralMemories.Lock()
		ephemeralMemories.m[lastMsgContext] = last
		ephemeralMemories.Unlock()
	}
}
