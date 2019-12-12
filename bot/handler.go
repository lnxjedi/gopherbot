package bot

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

	regexes.RLock()
	preRegex := regexes.preRegex
	postRegex := regexes.postRegex
	bareRegex := regexes.bareRegex
	regexes.RUnlock()
	currentCfg.RLock()
	ignoreUsers := currentCfg.ignoreUsers
	currentCfg.RUnlock()

	for _, user := range ignoreUsers {
		if strings.EqualFold(userName, user) {
			Log(robot.Debug, "Ignoring user", userName)
			emit(IgnoredUser)
			return
		}
	}
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

	globalTasks.Lock()
	t := globalTasks.t
	nameMap := globalTasks.nameMap
	idMap := globalTasks.idMap
	nameSpaces := globalTasks.nameSpaces
	globalTasks.Unlock()
	confLock.RLock()
	repolist := repositories
	confLock.RUnlock()
	currentCfg.RLock()
	cfg := currentCfg.configuration
	currentCfg.RUnlock()

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
		cfg:          cfg,
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

/* NOTE NOTE NOTE: Connector, Brain and History do not change after start-up, and that
probably shouldn't change. There's no real good reason to allow it, and not changing means
we don't need to worry about locking. When absolutely necessary, there's always "restart".
*/

// GetProtocolConfig unmarshals the connector's configuration data into a provided struct
func (h handler) GetProtocolConfig(v interface{}) error {
	err := json.Unmarshal(protocolConfig, v)
	return err
}

// GetBrainConfig unmarshals the brain's configuration data into a provided struct
func (h handler) GetBrainConfig(v interface{}) error {
	err := json.Unmarshal(brainConfig, v)
	return err
}

// GetHistoryConfig unmarshals the history provider's configuration data into a provided struct
func (h handler) GetHistoryConfig(v interface{}) error {
	err := json.Unmarshal(historyConfig, v)
	return err
}

// Log logs a message to the robot's log file (or stderr)
func (h handler) Log(l robot.LogLevel, m string, v ...interface{}) {
	Log(l, m, v...)
}

// GetDirectory verfies or creates a directory with perms 0750, returning an error on failure.
func (h handler) GetDirectory(p string) error {
	if len(p) == 0 {
		return errors.New("invalid 0-length path in GetDirectory")
	}
	dperm := os.FileMode(0750)
	if filepath.IsAbs(p) {
		p = filepath.Clean(p)
	}
	if ds, err := os.Stat(p); err == nil {
		if !ds.Mode().IsDir() {
			return fmt.Errorf("getting directory; '%s' exists but is not a directory", p)
		}
		if err := os.Chmod(p, dperm); err != nil {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(p, dperm); err != nil {
		return err
	}
	return nil
}

// SetBotID let's the connector set the bot's internal ID
func (h handler) SetBotID(id string) {
	currentCfg.Lock()
	currentCfg.botinfo.UserID = id
	currentCfg.Unlock()
}

// SetBotMention set's the @(mention) string, for regexes
func (h handler) SetBotMention(m string) {
	if len(m) == 0 {
		return
	}
	Log(robot.Info, "protocol set bot mention string to: %s", m)
	currentCfg.Lock()
	currentCfg.botinfo.protoMention = m
	currentCfg.Unlock()
	updateRegexes()
}
