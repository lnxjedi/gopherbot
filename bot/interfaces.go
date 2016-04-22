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
	// ChannelMessage is called by the connector for all messages the bot
	// can hear. The channelName and userName should be human-readable,
	// not internal representations.
	IncomingMessage(channelName, userName, message string)
	GetProtocolConfig(interface{}) error
	SetName(n string)
	GetLogLevel() LogLevel
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
