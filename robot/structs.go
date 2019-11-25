package robot

// AttrRet implements Stringer so it can be interpolated with fmt if
// the plugin author is ok with ignoring the RetVal.
type AttrRet struct {
	Attribute string
	RetVal
}

func (a AttrRet) String() string {
	return a.Attribute
}

// Message is passed to each task as it runs, initialized from the botContext.
// Tasks can copy and modify the Robot without affecting the botContext.
type Message struct {
	User            string            // The user who sent the message; this can be modified for replying to an arbitrary user
	ProtocolUser    string            // the protocol internal ID of the user
	Channel         string            // The channel where the message was received, or "" for a direct message. This can be modified to send a message to an arbitrary channel.
	ProtocolChannel string            // the protocol internal channel ID
	Protocol        Protocol          // slack, terminal, test, others; used for interpreting rawmsg or sending messages with Format = 'Raw'
	Incoming        *ConnectorMessage // raw struct of message sent by connector; interpret based on protocol. For Slack this is a *slack.MessageEvent
	Format          MessageFormat     // The outgoing message format, one of Raw, Fixed, or Variable
}

// ConnectorMessage is passed in to the robot for every incoming message seen.
// The *ID fields are required invariant internal representations that the
// protocol accepts in it's interface methods.
type ConnectorMessage struct {
	// Protocol - string name of connector, e.g. "Slack"
	Protocol string
	// optional UserName and required internal UserID
	UserName, UserID string
	// optional / required channel values
	ChannelName, ChannelID string
	// DirectMessage - whether the message should be considered private between user and robot
	DirectMessage bool
	// MessageText - sanitized message text, with all protocol-added junk removed
	MessageText string
	// MessageObject, Client - interfaces for the raw
	MessageObject, Client interface{}
}

// PluginHandler is the struct a plugin registers for the Gopherbot plugin API.
type PluginHandler struct {
	DefaultConfig string /* A yaml-formatted multiline string defining the default Plugin configuration. It should be liberally commented for use in generating
	custom configuration for the plugin. If a Config: section is defined, it should match the structure of the optional Config interface{} */
	Handler func(r Robot, command string, args ...string) TaskRetVal // The callback function called by the robot whenever a Command is matched
	Config  interface{}                                              // An optional empty struct defining custom configuration for the plugin
}

// PluginSpec used by loadable plugins that return a slice of PluginSpecs
type PluginSpec struct {
	Name    string
	Handler PluginHandler
}
