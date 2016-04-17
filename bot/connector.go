package bot

/* Definition of Connector interface, plus types and constants needed
by a Connector */

// type Connector is an interface that all protocols must implement
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
	// SendChannelMessage sends a message to a channel
	SendChannelMessage(channelname string, msg string)
	/* SendUserMessage sends a direct message to a user if supported.
	For protocols not supportint DM, the bot should send a message addressed
	to the user in an implementation-specific channel */
	SendUserMessage(username string, msg string)
}
