package bot

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
)

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
		b.lock.RLock()
		for _, adminUser := range b.adminUsers {
			if user == adminUser {
				isAdmin = true
				break
			}
		}
		b.lock.RUnlock()
		if !isAdmin {
			return false
		}
	}
	if len(plugin.Users) > 0 {
		userOk := false
		for _, allowedUser := range plugin.Users {
			if user == allowedUser {
				userOk = true
			}
		}
		if !userOk {
			return false
		}
	}
	if directMsg && !plugin.DisallowDirect {
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

// handleMessage checks the message against plugin commands and full-message matches,
// then dispatches it to all applicable handlers in a separate go routine. If the robot
// was addressed directly but nothing matched, any registered CatchAll plugins are called.
// There Should Be Only One
func handleMessage(isCommand bool, channel, user, messagetext string) {
	b.lock.RLock()
	bot := &Robot{
		User:    user,
		Channel: channel,
		Format:  Variable,
	}
	if len(channel) == 0 {
		Log(Trace, fmt.Sprintf("Bot received a direct message from %s: %s", user, messagetext))
	}
	commandMatched := false
	var catchAllPlugins []*Plugin
	if isCommand {
		catchAllPlugins = make([]*Plugin, 0, len(plugins))
	}
	// See if this is a reply that was requested
	matcher := replyMatcher{user, channel}
	botLock.Lock()
	if len(replies) > 0 {
		Log(Trace, fmt.Sprintf("Checking replies for matcher: %q", matcher))
		rep, exists := replies[matcher]
		if exists {
			Log(Debug, fmt.Sprintf("Found replyWaiter for user \"%s\" in channel \"%s\", checking message \"%s\" against \"%s\"", user, channel, messagetext, rep.re.String()))
			commandMatched = true
			// we got a match - so delete the matcher and send the reply struct
			delete(replies, matcher)
			matched := false
			if rep.re.MatchString(messagetext) {
				matched = true
			}
			rep.replyChannel <- reply{matched, messagetext}
		} else {
			Log(Trace, "No matching replyWaiter")
		}
	}
	botLock.Unlock()
	for _, plugin := range plugins {
		Log(Trace, fmt.Sprintf("Checking message \"%s\" against plugin %s, active in %d channels (allchannels: %t)", messagetext, plugin.name, len(plugin.Channels), plugin.AllChannels))
		ok := messageAppliesToPlugin(user, channel, plugin)
		if !ok {
			Log(Trace, fmt.Sprintf("Plugin %s ignoring message in channel %s, doesn't meet criteria", plugin.name, channel))
			continue
		}
		var matchers []InputMatcher
		if isCommand {
			matchers = plugin.CommandMatches
			if plugin.CatchAll {
				catchAllPlugins = append(catchAllPlugins, plugin)
			}
		} else {
			matchers = plugin.MessageMatches
		}
		for _, matcher := range matchers {
			Log(Trace, fmt.Sprintf("Checking \"%s\" against \"%s\"", messagetext, matcher.Regex))
			matches := matcher.re.FindAllStringSubmatch(messagetext, -1)
			if matches != nil {
				commandMatched = true
				privilegesOk := true
				if len(plugin.ElevatedCmds) > 0 {
					for _, i := range plugin.ElevatedCmds {
						if matcher.Command == i {
							if b.elevator != nil {
								privilegesOk = b.elevator(bot, false)
							} else {
								privilegesOk = false
								Log(Error, "Encountered elevated command and no elevation method configured")
							}
						}
					}
				}
				if len(plugin.ElevateImmediateCmds) > 0 {
					for _, i := range plugin.ElevateImmediateCmds {
						if matcher.Command == i {
							if b.elevator != nil {
								privilegesOk = b.elevator(bot, true)
							} else {
								privilegesOk = false
								Log(Error, "Encountered elevated command and no elevation method configured")
							}
						}
					}
				}
				if privilegesOk {
					go callPlugin(bot, plugin, matcher.Command, matches[0][1:]...)
				} else {
					Log(Error, fmt.Sprintf("Elevation failed for command \"%s\", plugin %s", matcher.Command, plugin.name))
					bot.Say(fmt.Sprintf("Sorry, the \"%s\" command requires elevated privileges", matcher.Command))
				}
			}
		}
	}
	if isCommand && !commandMatched { // the robot was spoken too, but nothing matched - call catchAlls
		for _, plugin := range catchAllPlugins {
			go callPlugin(bot, plugin, "catchall", messagetext)
		}
	}
	b.lock.RUnlock()
}

// callPlugin (normally called with go ...) sends a command to a plugin.
func callPlugin(bot *Robot, plugin *Plugin, command string, args ...string) {
	Log(Debug, fmt.Sprintf("Dispatching command \"%s\" to plugin \"%s\" with arguments \"%#v\"", command, plugin.name, args))
	bot.pluginID = plugin.pluginID
	switch plugin.pluginType {
	case plugBuiltin, plugGo:
		pluginHandlers[plugin.name].Handler(bot, command, args...)
	case plugExternal:
		var fullPath string // full path to the executable
		if len(plugin.pluginPath) == 0 {
			Log(Error, "pluginPath empty for external plugin:", plugin.name)
		}
		if byte(plugin.pluginPath[0]) == byte("/"[0]) {
			fullPath = plugin.pluginPath
		} else {
			_, err := os.Stat(b.localPath + "/" + plugin.pluginPath)
			if err != nil {
				_, err := os.Stat(b.installPath + "/" + plugin.pluginPath)
				if err != nil {
					Log(Error, fmt.Errorf("Couldn't locate external plugin %s: %v", plugin.name, err))
					return
				}
				fullPath = b.installPath + "/" + plugin.pluginPath
				Log(Debug, "Using stock external plugin:", fullPath)
			} else {
				fullPath = b.localPath + "/" + plugin.pluginPath
				Log(Debug, "Using local external plugin:", fullPath)
			}
		}
		externalArgs := make([]string, 0, 4+len(args))
		externalArgs = append(externalArgs, bot.Channel, bot.User, plugin.pluginID, command)
		externalArgs = append(externalArgs, args...)
		Log(Trace, fmt.Sprintf("Calling \"%s\" with args: %q", fullPath, externalArgs))
		// cmd := exec.Command(fullPath, channel, user, matcher.Command, matches[0][1:]...)
		cmd := exec.Command(fullPath, externalArgs...)
		// close stdout on the external plugin...
		cmd.Stdout = nil
		// but hold on to stderr in case we need to log an error
		stderr, err := cmd.StderrPipe()
		if err != nil {
			Log(Error, fmt.Errorf("Creating stderr pipe for external command \"%s\": %v", fullPath, err))
			bot.Reply(fmt.Sprintf("There were errors calling external plugin \"%s\", you might want to ask an administrator to check the logs", plugin.name))
			return
		}
		if err = cmd.Start(); err != nil {
			Log(Error, fmt.Errorf("Starting command \"%s\": %v", fullPath, err))
			bot.Reply(fmt.Sprintf("There were errors calling external plugin \"%s\", you might want to ask an administrator to check the logs", plugin.name))
			return
		}
		defer func() {
			if err = cmd.Wait(); err != nil {
				Log(Error, fmt.Errorf("Waiting on external command \"%s\": %v", fullPath, err))
				bot.Reply(fmt.Sprintf("There were errors calling external plugin \"%s\", you might want to ask an administrator to check the logs", plugin.name))
			}
		}()
		stdErrBytes, err := ioutil.ReadAll(stderr)
		if err != nil {
			Log(Error, fmt.Errorf("Reading from stderr for external command \"%s\": %v", fullPath, err))
			bot.Reply(fmt.Sprintf("There were errors calling external plugin \"%s\", you might want to ask an administrator to check the logs", plugin.name))
			return
		}
		stdErrString := string(stdErrBytes)
		if len(stdErrString) > 0 {
			Log(Warn, fmt.Errorf("Output from stderr of external command \"%s\": %s", fullPath, stdErrString))
			bot.Reply(fmt.Sprintf("There was error output while calling external plugin \"%s\", you might want to ask an administrator to check the logs", plugin.name))
		}
	}
}
