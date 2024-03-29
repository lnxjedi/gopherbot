package robot

import "io"

// Logger is used by a Brain for logging errors
type Logger interface {
	Log(l LogLevel, m string, v ...interface{})
}

// SimpleBrain is the simple interface for a configured brain, where the robot
// handles all locking issues.
type SimpleBrain interface {
	// Store stores a blob of data with a string key, returns error
	// if there's a problem storing the datum.
	Store(key string, blob *[]byte) error
	// Retrieve returns a blob of data (probably JSON) given a string key,
	// and exists=true if the data blob was found, or error if the brain
	// malfunctions.
	Retrieve(key string) (blob *[]byte, exists bool, err error)
	// List returns a list of all memories - Gopherbot isn't a database,
	// so it _should_ be pretty short.
	List() (keys []string, err error)
	// Delete deletes a memory
	Delete(key string) error
}

// Handler is the interface that defines the API for the handler object passed
// to Connectors, history providers and brain providers.
type Handler interface {
	// IncomingMessage is called by the connector for all messages the bot
	// can hear. See the fields for ConnectorMessage for information about
	// this object.
	IncomingMessage(*ConnectorMessage)
	// GetProtocolConfig unmarshals the ProtocolConfig section of robot.yaml
	// into a connector-provided struct
	GetProtocolConfig(interface{}) error
	// GetBrainConfig unmarshals the BrainConfig section of robot.yaml
	// into a struct provided by the brain provider
	GetBrainConfig(interface{}) error
	// GetEventStrings for developing tests with the terminal connector
	GetEventStrings() *[]string
	// GetHistoryConfig unmarshals the HistoryConfig section of robot.yaml
	// into a struct provided by the brain provider
	GetHistoryConfig(interface{}) error
	// SetID allows the connector to set the robot's internal ID
	SetBotID(id string)
	// SetTerminalWriter allows the terminal connector to provide an io.Writer
	// to log to.
	SetTerminalWriter(io.Writer)
	// SetBotMention allows the connector to set the bot's @(mention) ID
	// (without the @) for protocols where it's a fixed value. This allows
	// the robot to recognize "@(protoMention) foo", needed for e.g. Rocket
	// where the robot username may not match the configured name.
	SetBotMention(mention string)
	// GetLogLevel allows the connector to check the robot's configured log level
	// to make it's own decision about how much it should log. For slack, this
	// determines whether the plugin does api logging.
	GetLogLevel() LogLevel
	// GetInstallPath returns the installation path of the gopherbot
	GetInstallPath() string
	// GetConfigPath returns the path to the config directory if set
	GetConfigPath() string
	// Log provides a standard logging interface with a level as defined in
	// bot/logging.go
	Log(l LogLevel, m string, v ...interface{})
	// GetDirectory lets infrastructure plugins create directories, for e.g.
	// file-based history and brain providers. When privilege separation is in
	// use, the directory is created with the privileged uid.
	GetDirectory(path string) error
	// ExtractID is a convenience function for connectors, keeps 'import "regexp"
	// out of robot.
	ExtractID(u string) (string, bool)
	// RaisePriv raises the privilege of the current thread, allowing
	// filesystem access in GOPHER_HOME. Reason is informational.
	RaisePriv(reason string)
}

// Connector is the interface defining methods that should be provided by
// the connector for use by bot
type Connector interface {
	// SetUserMap provides the connector with a map from usernames to userIDs,
	// the protocol-internal ID for a user. The connector can use this map
	// to replace @name mentions in messages, and/or build a map of userIDs
	// to configured usernames.
	SetUserMap(map[string]string)
	// GetProtocolUserAttribute retrieves a piece of information about a user
	// from the connector protocol, or "",!ok if the connector doesn't have the
	// information. Plugins should normally call GetUserAttribute, which
	// supplements protocol data with data from users.json.
	// The connector should expect "username" or "<userid>".
	// The current attributes are:
	// email, realName, firstName, lastName, phone, sms, connections
	GetProtocolUserAttribute(user, attr string) (value string, ret RetVal)
	// MessageHeard tells the connector that the user should be notified that
	// the message has been heard and is being responded to. The connector
	// can then e.g. send a typing notifier.
	MessageHeard(user, channel string)
	// FormatHelp takes a (bot)/(alias) (command) - (description) string, and
	// returns a protocol-specific string formatted for display in the protocol.
	FormatHelp(string) string
	// DefaultHelp allows a connector to override the default help lines when
	// there is no keyword.
	DefaultHelp() []string
	// JoinChannel joins a channel given it's human-readable name, e.g. "general"
	JoinChannel(c string) RetVal
	/* NOTE: Each of the Send* methods takes a pointer to a ConnectorMessage.
	   For plugins, this is the original ConnectorMessage that triggered a
	   command, and provides context back to the connector in sending replies.
	*/
	// SendProtocolChannelThreadMessage sends a message to a thread in a channel,
	// starting a thread if none exists. If thread is unset or unsupported by the
	// protocol, it just sends a message to the channel.
	SendProtocolChannelThreadMessage(channelname, threadid, msg string, format MessageFormat, msgObject *ConnectorMessage) RetVal
	// SendProtocolUserChannelThreadMessage directs a message to a user in a channel/thread.
	// This method also supplies what the bot engine believes to be the username.
	SendProtocolUserChannelThreadMessage(userid, username, channelname, threadid, msg string, format MessageFormat, msgObject *ConnectorMessage) RetVal
	// SendProtocolUserMessage sends a direct message to a user if supported.
	// The value of user will be either "<userid>", the connector internal
	// userID in brackets, or "username", a string name the connector associates
	// with the user.
	SendProtocolUserMessage(user, msg string, format MessageFormat, msgObject *ConnectorMessage) RetVal
	// The Run method starts the main loop and takes a channel for stopping it.
	Run(stopchannel <-chan struct{})
}
