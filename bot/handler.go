package bot

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

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

// SetTerminalWriter let's the terminal connector set the output writer
// for logging of Warn and Error logs.
func (h handler) SetTerminalWriter(w io.Writer) {
	botStdOutLogger.SetOutput(w)
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

// A new worker is created for every incoming message, and may or may not end
// up creating a new pipeline. Workers are also created by scheduled jobs
// and Spawned jobs, in which case a pipeline is always created.
type worker struct {
	User            string                      // The user who sent the message; this can be modified for replying to an arbitrary user
	Channel         string                      // The channel where the message was received, or "" for a direct message. This can be modified to send a message to an arbitrary channel.
	ProtocolUser    string                      // The username or <userid> to be sent in connector methods
	ProtocolChannel string                      // the channel name or <channelid> where the message originated
	Protocol        robot.Protocol              // slack, terminal, test, others; used for interpreting rawmsg or sending messages with Format = 'Raw'
	Incoming        *robot.ConnectorMessage     // raw struct of message sent by connector; interpret based on protocol. For Slack this is a *slack.MessageEvent
	Format          robot.MessageFormat         // robot's default message format
	id              int                         // integer worker ID used when being registered as an active pipeline
	tasks           *taskList                   // Pointers to current task configuration at start of pipeline
	maps            *userChanMaps               // Pointer to current user / channel maps struct
	repositories    map[string]robot.Repository // Set of configured repositories
	cfg             *configuration              // Active configuration when this context was created
	BotUser         bool                        // set for bots/programs that should never match ambient messages
	listedUser      bool                        // set for users listed in the UserRoster; ambient messages don't match unlisted users by default
	isCommand       bool                        // Was the message directed at the robot, dm or by mention
	directMsg       bool                        // if the message was sent by DM
	msg             string                      // the message text sent
	automaticTask   bool                        // set for scheduled & triggers jobs, where user security restrictions don't apply
	*pipeContext                                // pointer to the pipeline context, created in
	sync.Mutex                                  // Lock to protect the bot context when pipeline running
}

// clone a worker for a new execution context
func (w *worker) clone() *worker {
	clone := &worker{
		User:            w.User,
		ProtocolUser:    w.ProtocolUser,
		Channel:         w.Channel,
		ProtocolChannel: w.ProtocolChannel,
		Incoming:        w.Incoming,
		directMsg:       w.directMsg,
		BotUser:         w.BotUser,
		listedUser:      w.listedUser,
		id:              getCtxID(),
		cfg:             w.cfg,
		tasks:           w.tasks,
		maps:            w.maps,
		repositories:    w.repositories,
		automaticTask:   w.automaticTask,
		Protocol:        w.Protocol,
		Format:          w.Format,
		msg:             w.msg,
	}
	if w.pipeContext != nil {
		w.Lock()
		clone.pipeContext = &pipeContext{
			pipeName:    w.pipeName,
			pipeDesc:    w.pipeDesc,
			ptype:       w.ptype,
			elevated:    w.elevated,
			environment: make(map[string]string),
		}
		w.Unlock()
	}
	return clone
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
	protocol, _ := getProtocol(inc.Protocol)

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
			Log(robot.Debug, ": %s", userName)
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

	currentCfg.RLock()
	cfg := currentCfg.configuration
	t := currentCfg.taskList
	currentCfg.RUnlock()

	confLock.RLock()
	repolist := repositories
	confLock.RUnlock()

	// Create the worker and a goroutine to process the message and carry state,
	// which may eventually run a pipeline.
	w := &worker{
		User:            userName,
		Channel:         channelName,
		ProtocolUser:    ProtocolUser,
		ProtocolChannel: ProtocolChannel,
		Protocol:        protocol,
		Incoming:        inc,
		Format:          cfg.defaultMessageFormat,
		tasks:           t,
		cfg:             cfg,
		maps:            maps,
		BotUser:         BotUser,
		listedUser:      listedUser,
		id:              getCtxID(),
		repositories:    repolist,
		isCommand:       isCommand,
		directMsg:       inc.DirectMessage,
		msg:             message,
	}
	if w.directMsg {
		Log(robot.Debug, "Received private message from user '%s'", userName)
	} else {
		Log(robot.Debug, "Message '%s' from user '%s' in channel '%s'; isCommand: %t", message, userName, logChannel, isCommand)
	}
	go w.handleMessage()
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
