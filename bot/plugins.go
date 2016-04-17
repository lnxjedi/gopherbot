package bot

// interface ChatBot defines the API for plugins
type ChatBot interface {
	Connector
	Log(l LogLevel, v ...interface{})
	GetLogLevel() LogLevel
	// SetLogLevel updates the connector log level
	SetLogLevel(l LogLevel)
}

// map from plugin names to handler functions
var goPluginHandlers map[string]func(bot ChatBot, channel, user, command string, args ...string) error = make(map[string]func(bot ChatBot, channel, user, command string, args ...string) error)

// RegisterPlugin allws plugins register a handler function in a func init().
// When the bot initializes, it will call each plugin's handler with a command
// "start" and no arguments, so the plugin can store a pointer to the bot
// object.
func RegisterPlugin(name string, handler func(bot ChatBot, channel, user, command string, args ...string) error) {
	goPluginHandlers[name] = handler
}
