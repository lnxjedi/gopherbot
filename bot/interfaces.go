package bot

/* Gather all the interfaces in once place. Structs should be defined
   close the their methods. */

// Logger is used by a Brain for logging errors
type Logger interface {
	Log(l LogLevel, v ...interface{})
}

// Handler is the interface that defines the callback API for Connectors
type Handler interface {
	// IncomingMessage is called by the connector for all messages the bot
	// can hear. The channelName and userName should be human-readable,
	// not internal representations. If channelName is blank, it's a direct message
	IncomingMessage(channelName, userName, message string)
	// GetProtocolConfig unmarshals the ProtocolConfig section of gopherbot.json
	// into a connector-provided struct
	GetProtocolConfig(interface{}) error
	// GetBrainConfig unmarshals the BrainConfig section of gopherbot.json
	// into a struct provided by the brain provider
	GetBrainConfig(interface{}) error
	// GetElevateConfig unmarshals the ElevateConfig section of gopherbot.json
	// into a struct provided by the elevate provider
	GetElevateConfig(interface{}) error
	// SetFullName allows the connector to set the robot's full name if it
	// has access to it.
	SetFullName(n string)
	// SetName allows the connect to set the robot's name that it should be addressed
	// by, if it has access to it.
	SetName(n string)
	// GetLogLevel allows the connector to check the robot's configured log level
	// to make it's own decision about how much it should log. For slack, this
	// determines whether the plugin does api logging.
	GetLogLevel() LogLevel
	// GetInstallPath returns the installation path of the gopherbot
	GetInstallPath() string
	// GetConfigPath returns the path to the local config of the gopherbot
	GetConfigPath() string
	// Log provides a standard logging interface with a level as defined in
	// bot/logging.go
	Log(l LogLevel, v ...interface{})
}

// SimpleBrain is the simple interface for a configured brain, where the robot
// handles all locking issues.
type SimpleBrain interface {
	// Store stores a blob of data with a string key, returns error
	// if there's a problem storing the datum.
	Store(key string, blob []byte) error
	// Retrieve returns a blob of data (probably JSON) given a string key,
	// and exists=true if the data blob was found, or error if the brain
	// malfunctions.
	Retrieve(key string) (blob []byte, exists bool, err error)
}

// Connector is the interface defining methods that should be provided by
// the connector for use by plugins/robot.
type Connector interface {
	// GetProtocolUserAttribute retrieves a piece of information about a user
	// from the connector protocol, or "",!ok if the connector doesn't have the
	// information. Plugins should normally call GetUserAttribute, which
	// supplements protocol data with data from users.json.
	// The current attributes are:
	// email, realName, firstName, lastName, phone, sms, connections
	GetProtocolUserAttribute(user, attr string) (value string, ret RetVal)
	// JoinChannel joins a channel given it's human-readable name, e.g. "general"
	JoinChannel(c string) RetVal
	// SendProtocolChannelMessage sends a message to a channel
	SendProtocolChannelMessage(channelname, msg string, format MessageFormat) RetVal
	// SendProtocolUserChannelMessage directs a message to a user in a channel
	SendProtocolUserChannelMessage(user, channelname, msg string, format MessageFormat) RetVal
	// SendProtocolUserMessage sends a direct message to a user if supported.
	// For protocols not supportint DM, the bot should send a message addressed
	// to the user in an implementation-specific channel.
	SendProtocolUserMessage(user, msg string, format MessageFormat) RetVal
	// The Run method starts the main loop, and never returns; if it's
	// called a second time, it just returns.
	Run(stopchannel chan struct{})
}
