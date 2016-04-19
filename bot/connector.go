package bot

/* Definition of Connector and Chatbot interfaces, giving all the methods
a connector needs to implement. */

// Chatbot is the interface defining methods that should be provided by
// the connector for use by plugins.
type Chatbot interface {
	// JoinChannel joins a channel given it's human-readable name, e.g. "general"
	JoinChannel(c string)
	// SendChannelMessage sends a message to a channel
	SendChannelMessage(channelname string, msg string)
	// SendUserMessage sends a direct message to a user if supported.
	// For protocols not supportint DM, the bot should send a message addressed
	// to the user in an implementation-specific channel.
	SendUserMessage(username string, msg string)
}

// type Connector is an interface that all protocols must implement
type Connector interface {
	Chatbot
	// GetProtocolUserAttribute retrieves a piece of information about a user
	// from the connector protocol, or "",!ok if the connector doesn't have the
	// information. Plugins should normally call GetUserAttribute, which
	// supplements protocol data with data from users.json.
	// The current attributes are:
	// email, realName, firstName, lastName, phone, sms
	GetProtocolUserAttribute(user, attr string) (value string, ok bool)
	// The Run method starts the main loop, and never returns
	Run()
}
