package bot

/* Gather all the interfaces in once place. Structs should be defined
   close the their methods. */

// Public interface for package main to initialize the robot with a connector
type GopherBot interface {
	GetConnectorName() string
	Init(c Connector)
	Handler // the Connector needs a Handler
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
	// Log provides a standard logging interface with a level as defined in
	// bot/logging.go
	Log(l LogLevel, v ...interface{})
}

// Chatbot is the interface defining methods that should be provided by
// the connector for use by plugins/robot.
type Connector interface {
	// GetProtocolUserAttribute retrieves a piece of information about a user
	// from the connector protocol, or "",!ok if the connector doesn't have the
	// information. Plugins should normally call GetUserAttribute, which
	// supplements protocol data with data from users.json.
	// The current attributes are:
	// email, realName, firstName, lastName, phone, sms
	GetProtocolUserAttribute(user, attr string) (value string, ok bool)
	// JoinChannel joins a channel given it's human-readable name, e.g. "general"
	JoinChannel(c string)
	// SendProtocolChannelMessage sends a message to a channel
	SendProtocolChannelMessage(channelname, msg string, format MessageFormat)
	// SendUserChannelMessage directs a message to a user in a channel
	SendProtocolUserChannelMessage(user, channelname, msg string, format MessageFormat)
	// SendProtocolUserMessage sends a direct message to a user if supported.
	// For protocols not supportint DM, the bot should send a message addressed
	// to the user in an implementation-specific channel.
	SendProtocolUserMessage(user, msg string, format MessageFormat)
	// The Run method starts the main loop, and never returns; if it's
	// called a second time, it just returns.
	Run()
}
