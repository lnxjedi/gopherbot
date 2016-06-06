package bot

// Default configuration for the robot expressed in yaml; gopherbot -dump <dir>
// will spit this out
const defaultConfig = `
# The administrator of this robot, can be retrieved in plugins with GetBotAttribute("admin")
AdminContact: ""
# If a plugin doesn't specify otherwise it will be active in these channels
DefaultChannels: [ "general", "random" ]
# Users the bot should never listen to
IgnoreUsers: []
# Note: Bot users in Slack can't join channels
JoinChannels: [ "random", "general" ]
# List of users that can issue admin commands like reload, quit
AdminUsers: []
# One-character alias the bot can be called by
Alias: ";"
# Port to listen on for http/JSON api calls, for external plugins
LocalPort: 8880
# One of trace, debug, info, warn, error
LogLevel: info
# List of external plugins to configure; generally scripts using a gopherbot script lib
ExternalPlugins:
- hosts
- echo
- whoami
Protocol: slack
ProtocolConfig:
  SlackToken: "<your_token_here>"
  # MaxMessageSplit specifies the maximum number of messages to break a message into when it's too long (>4000 char)
  MaxMessageSplit: 2
# Specify the mechanism for storing the robots memories
Brain: file
BrainConfig:
  # For a file-storage based brain, just store files in a directory. Relative paths are relative to the local config dir.
  BrainDirectory: brain
`
