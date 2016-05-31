package bot

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
)

// a replyWaiter is used when a plugin is waiting for a reply
type replyWaiter struct {
	regex        string         // The text of the regular expression
	re           *regexp.Regexp // The regular expression the reply needs to match
	replyChannel chan reply     // The channel to send the reply to when it is received
}

// a reply matcher is used as the key in the replys map
type replyMatcher struct {
	user, channel string // Only one reply at a time can be requested for a given user/channel combination
}

// a reply is sent over the replyWaiter channel when a user replies
type reply struct {
	matched bool   // true if the regex matched
	rep     string // text of the reply if matched=true, else ""
}

var replies = make(map[replyMatcher]replyWaiter)

// messageAppliesToPlugin checks the user and channel against the plugin's
// configuration to determine if the message should be evaluated. Used by
// both handleMessage and the help builtin.
func (b *robot) messageAppliesToPlugin(user, channel, message string, plugin Plugin) bool {
	directMsg := false
	if len(channel) == 0 {
		directMsg = true
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
func (b *robot) handleMessage(isCommand bool, channel, user, messagetext string) {
	b.lock.RLock()
	bot := Robot{
		User:    user,
		Channel: channel,
		Format:  Variable,
		robot:   b,
	}
	if len(channel) == 0 {
		b.Log(Trace, fmt.Sprintf("Bot received a direct message from %s: %s", user, messagetext))
	}
	commandMatched := false
	var catchAllPlugins []Plugin
	if isCommand {
		catchAllPlugins = make([]Plugin, 0, len(plugins))
	}
	// See if this is a reply that was requested
	matcher := replyMatcher{user, channel}
	botLock.Lock()
	if len(replies) > 0 {
		b.Log(Trace, fmt.Sprintf("Checking replies for matcher: %q", matcher))
		rep, exists := replies[matcher]
		if exists {
			b.Log(Debug, fmt.Sprintf("Found replyWaiter for user \"%s\" in channel \"%s\", checking message \"%s\" against \"%s\"", user, channel, messagetext, rep.regex))
			commandMatched = true
			// we got a match - so delete the matcher and send the reply struct
			delete(replies, matcher)
			matched := false
			if rep.re.MatchString(messagetext) {
				matched = true
			}
			rep.replyChannel <- reply{matched, messagetext}
		} else {
			b.Log(Trace, "No matching replyWaiter")
		}
	}
	botLock.Unlock()
	for _, plugin := range plugins {
		b.Log(Trace, fmt.Sprintf("Checking message \"%s\" against plugin %s, active in %d channels", messagetext, plugin.Name, len(plugin.Channels)))
		ok := b.messageAppliesToPlugin(user, channel, messagetext, plugin)
		if !ok {
			b.Log(Trace, fmt.Sprintf("Plugin %s ignoring message in channel %s, doesn't meet criteria", plugin.Name, channel))
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
			b.Log(Trace, fmt.Sprintf("Checking \"%s\" against \"%s\"", messagetext, matcher.Regex))
			matches := matcher.re.FindAllStringSubmatch(messagetext, -1)
			if matches != nil {
				commandMatched = true
				go b.callPlugin(bot, plugin, matcher.Command, matches[0][1:]...)
			}
		}
	}
	if isCommand && !commandMatched { // the robot was spoken too, but nothing matched - call catchAlls
		for _, plugin := range catchAllPlugins {
			go b.callPlugin(bot, plugin, "catchall", messagetext)
		}
	}
	b.lock.RUnlock()
}

// callPlugin (normally called with go ...) sends a command to a plugin.
func (b *robot) callPlugin(bot Robot, plugin Plugin, command string, args ...string) {
	b.Log(Debug, fmt.Sprintf("Dispatching command %s to plugin %s", command, plugin.Name))
	bot.pluginID = plugin.pluginID
	switch plugin.pluginType {
	case plugBuiltin, plugGo:
		pluginHandlers[plugin.Name].Handler(bot, command, args...)
	case plugExternal:
		var fullPath string // full path to the executable
		if len(plugin.PluginPath) == 0 {
			b.Log(Error, "PluginPath empty for external plugin:", plugin.Name)
		}
		if byte(plugin.PluginPath[0]) == byte("/"[0]) {
			fullPath = plugin.PluginPath
		} else {
			_, err := os.Stat(b.localPath + "/" + plugin.PluginPath)
			if err != nil {
				_, err := os.Stat(b.installPath + "/" + plugin.PluginPath)
				if err != nil {
					b.Log(Error, fmt.Errorf("Couldn't locate external plugin %s: %v", plugin.Name, err))
					return
				}
				fullPath = b.installPath + "/" + plugin.PluginPath
				b.Log(Debug, "Using stock external plugin:", fullPath)
			} else {
				fullPath = b.localPath + "/" + plugin.PluginPath
				b.Log(Debug, "Using local external plugin:", fullPath)
			}
		}
		externalArgs := make([]string, 0, 4+len(args))
		externalArgs = append(externalArgs, bot.Channel, bot.User, plugin.pluginID, command)
		externalArgs = append(externalArgs, args...)
		b.Log(Trace, fmt.Sprintf("Calling \"%s\" with args: %q", fullPath, externalArgs))
		// cmd := exec.Command(fullPath, channel, user, matcher.Command, matches[0][1:]...)
		cmd := exec.Command(fullPath, externalArgs...)
		// close stdout on the external plugin...
		cmd.Stdout = nil
		// but hold on to stderr in case we need to log an error
		stderr, err := cmd.StderrPipe()
		if err != nil {
			b.Log(Error, fmt.Errorf("Creating stderr pipe for external command \"%s\": %v", fullPath, err))
			return
		}
		if err := cmd.Start(); err != nil {
			b.Log(Error, fmt.Errorf("Starting command \"%s\": %v", fullPath, err))
			return
		}
		defer func() {
			if err := cmd.Wait(); err != nil {
				b.Log(Error, fmt.Errorf("Waiting on external command \"%s\": %v", fullPath, err))
			}
		}()
		stdErrBytes, err := ioutil.ReadAll(stderr)
		if err != nil {
			b.Log(Error, fmt.Errorf("Reading from stderr for external command \"%s\": %v", fullPath, err))
			return
		}
		stdErrString := string(stdErrBytes)
		if len(stdErrString) > 0 {
			b.Log(Warn, fmt.Errorf("Output from stderr of external command \"%s\": %s", fullPath, stdErrString))
		}
	}
}
