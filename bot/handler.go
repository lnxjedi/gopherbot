package bot

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lnxjedi/gopherbot/robot"
	"gopkg.in/yaml.v3"
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
	botStdErrLogger.SetOutput(w)
	terminalWriter = w
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

// RaisePriv raises privilege for connectors, brains, etc.
func (h handler) RaisePriv(reason string) {
	raiseThreadPriv(reason)
}

// A new worker is created for every incoming message, and may or may not end
// up creating a new pipeline. Workers are also created by scheduled jobs
// and Spawned jobs, in which case a pipeline is always created.
type worker struct {
	User            string                  // The user who sent the message; this can be modified for replying to an arbitrary user
	Channel         string                  // The channel where the message was received, or "" for a direct message. This can be modified to send a message to an arbitrary channel.
	ProtocolUser    string                  // The username or <userid> to be sent in connector methods
	ProtocolChannel string                  // the channel name or <channelid> where the message originated
	Protocol        robot.Protocol          // slack, terminal, test, others; used for interpreting rawmsg or sending messages with Format = 'Raw'
	Incoming        *robot.ConnectorMessage // raw struct of message sent by connector
	Format          robot.MessageFormat     // robot's default message format
	id              int                     // integer worker ID used when being registered as an active pipeline
	tasks           *taskList               // Pointers to current task configuration at start of pipeline
	maps            *userChanMaps           // Pointer to current user / channel maps struct
	cfg             *configuration          // Active configuration when this context was created
	BotUser         bool                    // set for bots/programs that should never match ambient messages
	listedUser      bool                    // set for users listed in the UserRoster; ambient messages don't match unlisted users by default
	isCommand       bool                    // Was the message directed at the robot, dm or by mention
	cmdMode         string                  // one of "alias", "name", "direct" - for disambiguation
	msg, fmsg       string                  // the message text sent; without robot name/alias, and with for message matching
	automaticTask   bool                    // set for scheduled & triggers jobs, where user security restrictions don't apply
	*pipeContext                            // pointer to the pipeline context
	sync.Mutex                              // Lock to protect the bot context when pipeline running
}

// clone a worker for a new execution context
func (w *worker) clone() *worker {
	clone := &worker{
		User:            w.User,
		ProtocolUser:    w.ProtocolUser,
		Channel:         w.Channel,
		ProtocolChannel: w.ProtocolChannel,
		Incoming:        w.Incoming,
		BotUser:         w.BotUser,
		listedUser:      w.listedUser,
		id:              getWorkerID(),
		cfg:             w.cfg,
		tasks:           w.tasks,
		maps:            w.maps,
		automaticTask:   w.automaticTask,
		Protocol:        w.Protocol,
		Format:          w.Format,
		msg:             w.msg,
		fmsg:            w.fmsg,
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

// for logging raw messages from channels
var chanLoggers = struct {
	sync.Mutex
	channels map[string]*log.Logger
}{
	Mutex:    sync.Mutex{},
	channels: make(map[string]*log.Logger),
}

// ChannelMessage accepts an incoming channel message from the connector.
// func (h handler) IncomingMessage(channelName, userName, messageFull string, raw interface{}) {
func (h handler) IncomingMessage(inc *robot.ConnectorMessage) {
	// Note: zero-len channel name and ID is valid; true of direct messages for some connectors
	if len(inc.UserName) == 0 && len(inc.UserID) == 0 {
		Log(robot.Error, "Incoming message with no username or user ID")
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
	protocol := getProtocol(inc.Protocol)

	messageFull := inc.MessageText
	var message string

	chanLoggers.Lock()
	clog, ok := chanLoggers.channels[channelName]
	chanLoggers.Unlock()
	if ok {
		clog.Printf("c:%s/%s(m:%s/t:%s/%t), u:%s/%s, m:'%s'\n", channelName, ProtocolChannel, inc.MessageID, inc.ThreadID, inc.ThreadedMessage, userName, ProtocolUser, messageFull)
	}
	Log(robot.Trace, "Incoming message in channel '%s/%s' from user '%s/%s': %s", channelName, ProtocolChannel, userName, ProtocolUser, messageFull)
	// When command == true, the message was directed at the bot
	isCommand := false
	logChannel := channelName

	currentCfg.RLock()
	ignoreUsers := currentCfg.ignoreUsers
	ignoreUnlisted := currentCfg.ignoreUnlistedUsers
	botAlias := currentCfg.alias
	currentCfg.RUnlock()
	if !listedUser && ignoreUnlisted {
		Log(robot.Debug, "IgnoreUnlistedUsers - ignoring: %s / %s", inc.UserID, userName)
		emit(IgnoredUser)
		return
	}
	for _, user := range ignoreUsers {
		if strings.EqualFold(userName, user) {
			Log(robot.Debug, "User listed in IgnoreUsers, ignoring: %s", userName)
			emit(IgnoredUser)
			return
		}
	}
	cmdMode := ""
	// Log(robot.Debug, "DEBUG: incoming %+v", inc)
	if inc.BotMessage {
		isCommand = true
		message = messageFull
		cmdMode = "alias" // technically not true, but makes more sense
	} else {
		regexes.RLock()
		preRegex := regexes.preRegex
		postRegex := regexes.postRegex
		bareRegex := regexes.bareRegex
		regexes.RUnlock()
		if preRegex != nil {
			matches := preRegex.FindStringSubmatch(messageFull)
			if len(matches) == 3 {
				isCommand = true
				name := matches[1]
				if name == string(botAlias) {
					cmdMode = "alias"
				} else {
					cmdMode = "name"
				}
				message = matches[2]
			}
		}
		if !isCommand && postRegex != nil {
			matches := postRegex.FindStringSubmatch(messageFull)
			if len(matches) == 3 {
				isCommand = true
				cmdMode = "name"
				message = matches[1] + matches[2]
			}
		}
		if !isCommand && bareRegex != nil {
			if bareRegex.MatchString(messageFull) {
				if messageFull == string(botAlias) {
					cmdMode = "alias"
				} else {
					cmdMode = "name"
				}
				isCommand = true
			}
		}
		if !isCommand {
			message = messageFull
		}
	}

	if inc.DirectMessage {
		isCommand = true
		// When the user addresses the robot by name or alias in a DM,
		// we stick with that, otherwise we use "direct" so the
		// plugin can try to be smart.
		if len(cmdMode) == 0 {
			cmdMode = "direct"
		}
		logChannel = "(direct message)"
		// We don't support threads in DMs; we blank out the thread
		// so replies will match.
		inc.ThreadID = ""
	}

	currentCfg.RLock()
	cfg := currentCfg.configuration
	t := currentCfg.taskList
	currentCfg.RUnlock()

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
		id:              getWorkerID(),
		isCommand:       isCommand,
		cmdMode:         cmdMode,
		msg:             message,
		fmsg:            messageFull,
	}
	if w.Incoming.DirectMessage {
		Log(robot.Debug, "Received private message from user '%s'", userName)
	} else {
		Log(robot.Debug, "Message '%s'/id '%s' from user '%s' in channel '%s'/thread '%s' (threaded: %t); isCommand: %t; cmdMode: %s", message, inc.MessageID, userName, logChannel, inc.ThreadID, inc.ThreadedMessage, isCommand, cmdMode)
	}
	go w.handleMessage()
}

/* NOTE NOTE NOTE: Connector, Brain and History do not change after start-up, and that
probably shouldn't change. There's no real good reason to allow it, and not changing means
we don't need to worry about locking. When absolutely necessary, there's always "restart".
*/

// GetProtocolConfig unmarshals the connector's configuration data into a provided struct
func (h handler) GetProtocolConfig(v interface{}) error {
	if protocolConfig != nil {
		data, err := yaml.Marshal(protocolConfig)
		if err != nil {
			return fmt.Errorf("marshaling ProtocolConfig: %v", err)
		}
		if err := yaml.Unmarshal(data, v); err != nil {
			return fmt.Errorf("unmarshaling ProtocolConfig into provided struct: %v", err)
		}
	}
	return nil
}

// GetBrainConfig unmarshals the brain's configuration data into a provided struct
func (h handler) GetBrainConfig(v interface{}) error {
	if brainConfig != nil {
		data, err := yaml.Marshal(brainConfig)
		if err != nil {
			return fmt.Errorf("marshaling BrainConfig: %v", err)
		}
		if err := yaml.Unmarshal(data, v); err != nil {
			return fmt.Errorf("unmarshaling BrainConfig into provided struct: %v", err)
		}
	}
	return nil
}

// GetHistoryConfig unmarshals the history provider's configuration data into a provided struct
func (h handler) GetHistoryConfig(v interface{}) error {
	if historyConfig != nil {
		data, err := yaml.Marshal(historyConfig)
		if err != nil {
			return fmt.Errorf("marshaling HistoryConfig: %v", err)
		}
		if err := yaml.Unmarshal(data, v); err != nil {
			return fmt.Errorf("unmarshaling HistoryConfig into provided struct: %v", err)
		}
	}
	return nil
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
	raiseThreadPriv(fmt.Sprintf("getting directory: %s", p))
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
	Log(robot.Info, "Protocol set bot mention string to: %s", m)
	currentCfg.Lock()
	currentCfg.botinfo.protoMention = m
	currentCfg.Unlock()
	updateRegexes()
}
