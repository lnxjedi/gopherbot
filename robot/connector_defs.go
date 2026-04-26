package robot

import "io"

// Protocol - connector protocols
type Protocol int

const (
	// Slack connector
	Slack Protocol = iota
	// Google Chat connector
	GoogleChat
	// Rocket for Rocket.Chat
	Rocket
	// Terminal connector
	Terminal
	// Test connector for automated test suites
	Test
	// Null connector for unconfigured robots
	Null
	// SSH connector for local development
	SSH
)

// ConnectorMessage is passed in to the robot for every incoming message seen.
// The *ID fields are required invariant internal representations that the
// protocol accepts in it's interface methods.
type ConnectorMessage struct {
	// Protocol - string name of connector, e.g. "Slack"
	Protocol string
	// optional UserName and required internal UserID
	UserName, UserID string
	// ValidatedUser - true when the connector can vouch that UserName is the
	// correct canonical username for this specific UserID.
	ValidatedUser bool
	// optional / required channel values
	ChannelName, ChannelID string
	// Opaque values
	ThreadID, MessageID string
	ThreadedMessage     bool
	// true when the incoming message originated from the robot itself
	SelfMessage bool
	// DirectMessage - whether the message should be considered private between user and robot
	DirectMessage bool
	// BotMessage - true when the connector is certain the message has been sent to the robot,
	// e.g. for slack slash commands
	BotMessage bool
	// HiddenMessage - true when the user sent a message to the robot that can't be seen by
	// other users, also true for slack slash commands
	HiddenMessage bool
	// MessageText - sanitized message text, with all protocol-added junk removed
	MessageText string
	// MessageObject, Client - interfaces for the raw objects; go extensions can use
	// these with type switches/assertions to access object internals
	MessageObject, Client interface{}
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
	// GetBotInfo returns the configured robot identity from robot.yaml.
	GetBotInfo() BotInfo
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
	// ReadEncryptedFile reads and decrypts a Gopherbot-encrypted file from the
	// robot config/install area, returning the plaintext bytes.
	ReadEncryptedFile(path string) ([]byte, error)
	// Log provides a standard logging interface with a level as defined in
	// bot/logging.go
	Log(l LogLevel, m string, v ...interface{})
	// GetDirectory lets infrastructure plugins create directories, for e.g.
	// file-based history and brain providers. When privilege separation is in
	// use, the directory is created with the privileged uid.
	GetDirectory(path string) error
	// RaisePriv raises the privilege of the current thread, allowing
	// filesystem access in GOPHER_HOME. Reason is informational.
	RaisePriv(reason string)
}

// Connector is the interface defining methods that should be provided by
// the connector for use by bot
type Connector interface {
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
	// SendProtocolUserChannelThreadMessage directs a message to a user in a
	// channel/thread.
	// userid carries the protocol-local target identity when one exists, usually
	// as a bracketed internal ID like "<U123...>".
	// username carries the engine's canonical user identity for readable fallback
	// formatting or protocol-local lookup.
	SendProtocolUserChannelThreadMessage(userid, username, channelname, threadid, msg string, format MessageFormat, msgObject *ConnectorMessage) RetVal
	// SendProtocolUserMessage sends a direct message to a user if supported.
	// The value of user is the engine's canonical username identity.
	SendProtocolUserMessage(user, msg string, format MessageFormat, msgObject *ConnectorMessage) RetVal
	// Reload refreshes connector-local runtime configuration after a
	// successful engine configuration reload. Implementations must apply
	// changes atomically so live readers see either the old complete connector
	// state or the new complete connector state.
	Reload() error
	// The Run method starts the main loop and takes a channel for stopping it.
	Run(stopchannel <-chan struct{})
}

// ConnectorAPIProvider allows a connector to expose optional connector-specific
// APIs without widening the base Connector interface.
type ConnectorAPIProvider interface {
	ConnectorAPI() interface{}
}

// InjectMessageRequest describes an injected inbound message for connector API users.
type InjectMessageRequest struct {
	AsUser  string
	Text    string
	Channel string
	Thread  string
	Hidden  bool
	Direct  bool
}

// InjectMessageResult describes the connector's accepted/injected message.
type InjectMessageResult struct {
	Protocol  string
	UserName  string
	UserID    string
	Channel   string
	MessageID string
	ThreadID  string
	Hidden    bool
	Direct    bool
	Cursor    uint64
	Timestamp string
}

// MessageQuery requests connector messages for a viewer/cursor.
type MessageQuery struct {
	Viewer      string
	AfterCursor uint64
	Limit       int
	TimeoutMS   int
	All         bool
}

// MessageEvent is a connector message item suitable for polling/cursor retrieval.
type MessageEvent struct {
	Cursor    uint64
	Timestamp string
	UserName  string
	UserID    string
	IsBot     bool
	Channel   string
	ThreadID  string
	MessageID string
	Threaded  bool
	Text      string
	Direct    bool
	Hidden    bool
}

// MessageBatch is a batch response for message polling/cursor retrieval.
type MessageBatch struct {
	Protocol   string
	Viewer     string
	Messages   []MessageEvent
	NextCursor uint64
	Latest     uint64
	TimedOut   bool
	Overflow   bool
	HasMore    bool
}

// Injector is an optional connector API for injecting inbound messages.
type Injector interface {
	InjectMessage(req InjectMessageRequest) (InjectMessageResult, error)
}

// MessageSource is an optional connector API for cursor-based message retrieval.
type MessageSource interface {
	GetMessages(req MessageQuery) (MessageBatch, error)
}
