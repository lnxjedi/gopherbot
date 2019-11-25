package bot

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

// an empty object type for passing a Handler to the connector.
type handler struct{}

// dummy var to pass a handler
var handle = handler{}

/* Handle incoming messages and other callbacks from the connector. */

// GetLogLevel returns the bot's current loglevel, mainly for the
// connector to make it's own decision about logging
func (h handler) GetLogLevel() robot.LogLevel {
	return getLogLevel()
}

// GetLogToFile indicates to the terminal connector whether logging output is
// to a file, to prevent readline from redirecting log output.
func (h handler) GetLogToFile() bool {
	return logToFile
}

// GetInstallPath gets the path to the bot's install dir -
// the location of default configuration and stock external plugins.
func (h handler) GetInstallPath() string {
	return installPath
}

// GetConfigPath gets the path to the bot's (supposedly writable) configuration
// directory. This is the config path if specified, otherwise the install
// directory.
func (h handler) GetConfigPath() string {
	if len(configPath) > 0 {
		return configPath
	}
	return installPath
}

// ChannelMessage accepts an incoming channel message from the connector.
//func (h handler) IncomingMessage(channelName, userName, messageFull string, raw interface{}) {
func (h handler) IncomingMessage(inc *robot.ConnectorMessage) {
	// Note: zero-len channel name and ID is valid; true of direct messages for some connectors
	if len(inc.UserName) == 0 && len(inc.UserID) == 0 {
		Log(robot.Error, "incoming message with no username or user ID")
		return
	}
	currentUCMaps.Lock()
	maps := currentUCMaps.ucmap
	currentUCMaps.Unlock()
	var channelName, userName, ProtocolChannel, ProtocolUser string
	var BotUser bool

	/* Make sure some form of User and Channel are set
	 */
	ProtocolChannel = bracket(inc.ChannelID)
	if !inc.DirectMessage {
		if cn, ok := maps.channelID[inc.ChannelID]; ok {
			channelName = cn.ChannelName
		} else if len(inc.ChannelName) > 0 {
			channelName = inc.ChannelName
		} else if len(inc.ChannelID) > 0 {
			channelName = bracket(inc.ChannelID)
		}
	} // ProtocolChannel / channelName should be "" for DM
	ProtocolUser = bracket(inc.UserID)
	listedUser := false
	if un, ok := maps.userID[inc.UserID]; ok {
		userName = un.UserName
		BotUser = un.BotUser
		listedUser = true
	} else if len(inc.UserName) > 0 {
		userName = inc.UserName
	} else {
		userName = bracket(inc.UserID)
	}

	messageFull := inc.MessageText

	Log(robot.Trace, "Incoming message in channel '%s/%s' from user '%s/%s': %s", channelName, ProtocolChannel, userName, ProtocolUser, messageFull)
	// When command == true, the message was directed at the bot
	isCommand := false
	logChannel := channelName
	var message string

	botCfg.RLock()
	for _, user := range botCfg.ignoreUsers {
		if strings.EqualFold(userName, user) {
			Log(robot.Debug, "Ignoring user", userName)
			c := &botContext{User: userName}
			c.debug("robot is configured to ignore this user", true)
			emit(IgnoredUser)
			botCfg.RUnlock()
			return
		}
	}
	preRegex := botCfg.preRegex
	postRegex := botCfg.postRegex
	bareRegex := botCfg.bareRegex
	botCfg.RUnlock()
	if preRegex != nil {
		matches := preRegex.FindAllStringSubmatch(messageFull, -1)
		if matches != nil && len(matches[0]) == 2 {
			isCommand = true
			message = matches[0][1]
		}
	}
	if !isCommand && postRegex != nil {
		matches := postRegex.FindAllStringSubmatch(messageFull, -1)
		if matches != nil && len(matches[0]) == 3 {
			isCommand = true
			message = matches[0][1] + matches[0][2]
		}
	}
	if !isCommand && bareRegex != nil {
		if bareRegex.MatchString(messageFull) {
			isCommand = true
		}
	}
	if !isCommand {
		message = messageFull
	}

	if inc.DirectMessage {
		isCommand = true
		logChannel = "(direct message)"
	}

	currentTasks.Lock()
	t := currentTasks.t
	nameMap := currentTasks.nameMap
	idMap := currentTasks.idMap
	nameSpaces := currentTasks.nameSpaces
	currentTasks.Unlock()
	confLock.RLock()
	repolist := repositories
	confLock.RUnlock()

	// Create the botContext and a goroutine to process the message and carry state,
	// which may eventually run a pipeline.
	c := &botContext{
		User:            userName,
		Channel:         channelName,
		ProtocolUser:    ProtocolUser,
		ProtocolChannel: ProtocolChannel,
		Incoming:        inc,
		tasks: taskList{
			t:          t,
			nameMap:    nameMap,
			idMap:      idMap,
			nameSpaces: nameSpaces,
		},
		maps:         maps,
		BotUser:      BotUser,
		listedUser:   listedUser,
		repositories: repolist,
		isCommand:    isCommand,
		directMsg:    inc.DirectMessage,
		msg:          message,
		environment:  make(map[string]string),
	}
	if c.directMsg {
		Log(robot.Debug, "Received private message from user '%s'", userName)
	} else {
		Log(robot.Debug, "Message '%s' from user '%s' in channel '%s'; isCommand: %t", message, userName, logChannel, isCommand)
		c.debug(fmt.Sprintf("Message (command: %v) in channel %s: %s", isCommand, logChannel, message), true)
	}
	go c.handleMessage()
}

// GetProtocolConfig unmarshals the connector's configuration data into a provided struct
func (h handler) GetProtocolConfig(v interface{}) error {
	botCfg.RLock()
	err := json.Unmarshal(protocolConfig, v)
	botCfg.RUnlock()
	return err
}

// GetBrainConfig unmarshals the brain's configuration data into a provided struct
func (h handler) GetBrainConfig(v interface{}) error {
	botCfg.RLock()
	err := json.Unmarshal(brainConfig, v)
	botCfg.RUnlock()
	return err
}

// GetHistoryConfig unmarshals the history provider's configuration data into a provided struct
func (h handler) GetHistoryConfig(v interface{}) error {
	botCfg.RLock()
	err := json.Unmarshal(historyConfig, v)
	botCfg.RUnlock()
	return err
}

// Log logs a message to the robot's log file (or stderr)
func (h handler) Log(l robot.LogLevel, m string, v ...interface{}) {
	Log(l, m, v...)
}

// SetBotID let's the connector set the bot's internal ID
func (h handler) SetBotID(id string) {
	botCfg.Lock()
	botCfg.botinfo.UserID = id
	botCfg.Unlock()
}

// SetBotMention set's the @(mention) string, for regexes
func (h handler) SetBotMention(m string) {
	if len(m) == 0 {
		return
	}
	Log(robot.Info, "protocol set bot mention string to: %s", m)
	botCfg.Lock()
	botCfg.botinfo.protoMention = m
	botCfg.Unlock()
	updateRegexes()
}
