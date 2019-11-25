package robot

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
}

// Handler is the interface that defines the callback API for Connectors
type Handler interface {
	// IncomingMessage is called by the connector for all messages the bot
	// can hear. See the fields for ConnectorMessage for information about
	// this object.
	IncomingMessage(*ConnectorMessage)
	// GetProtocolConfig unmarshals the ProtocolConfig section of gopherbot.yaml
	// into a connector-provided struct
	GetProtocolConfig(interface{}) error
	// GetBrainConfig unmarshals the BrainConfig section of gopherbot.yaml
	// into a struct provided by the brain provider
	GetBrainConfig(interface{}) error
	// GetHistoryConfig unmarshals the HistoryConfig section of gopherbot.yaml
	// into a struct provided by the brain provider
	GetHistoryConfig(interface{}) error
	// SetID allows the connector to set the robot's internal ID
	SetBotID(id string)
	// SetBotMention allows the connector to set the bot's @(mention) ID
	// (without the @) for protocols where it's a fixed value. This allows
	// the robot to recognize "@(protoMention) foo", needed for e.g. Rocket
	// where the robot username may not match the configured name.
	SetBotMention(mention string)
	// GetLogLevel allows the connector to check the robot's configured log level
	// to make it's own decision about how much it should log. For slack, this
	// determines whether the plugin does api logging.
	GetLogLevel() LogLevel
	// GetLogToFile is for the terminal connector to determine if logging is
	// going to a file, to prevent readline from redirecting log output.
	GetLogToFile() bool
	// GetInstallPath returns the installation path of the gopherbot
	GetInstallPath() string
	// GetConfigPath returns the path to the config directory if set
	GetConfigPath() string
	// Log provides a standard logging interface with a level as defined in
	// bot/logging.go
	Log(l LogLevel, m string, v ...interface{})
	// Convenience function for connectors, keeps 'import "regexp" out of
	// robot.
	ExtractID(u string) (string, bool)
}

// Connector is the interface defining methods that should be provided by
// the connector for use by plugins/bot
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
	// JoinChannel joins a channel given it's human-readable name, e.g. "general"
	JoinChannel(c string) RetVal
	// SendProtocolChannelMessage sends a message to a channel
	SendProtocolChannelMessage(channelname, msg string, format MessageFormat) RetVal
	// SendProtocolUserChannelMessage directs a message to a user in a channel
	// This method also supplies what the bot engine believes to be the username.
	SendProtocolUserChannelMessage(userid, username, channelname, msg string, format MessageFormat) RetVal
	// SendProtocolUserMessage sends a direct message to a user if supported.
	// The value of user will be either "<userid>", the connector internal
	// userID in brackets, or "username", a string name the connector associates
	// with the user.
	SendProtocolUserMessage(user, msg string, format MessageFormat) RetVal
	// The Run method starts the main loop and takes a channel for stopping it.
	Run(stopchannel <-chan struct{})
}
