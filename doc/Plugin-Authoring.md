**NOTE: Unfinished**

This article deals mainly with writing plugins in one of the scripting languages supported by Gopherbot,
the most popular means for writing new command plugins. For writing native compiled-in plugins in Go, see
`gopherbot/main.go` and the sample plugins in `goplugins/`. API documentation for Robot methods is available
at:

https://godoc.org/github.com/uva-its/gopherbot/bot#Robot

Note that the script plugin API sits on top of the native Go API, so that document is also of use for scripting plugin authors.

# Configuring Plugins
Plugin configuration is fully documented in the [configuration](Configuration.md) article, and you should
read that document before starting to write your own plugins.

# Getting Started
The best way for a new plugin author to get started is to:
* Disable the demo plugin for your chosen scripting language (if enabled) in `<config dir>/conf/gopherbot.yaml`
* Copy the demo plugin to `<config dir>/plugins/<newname>(.extension)`
* Enable your new plugin in `gopherbot.yaml` and give it a descriptive `Name`

# Calling Conventions

** reserved commands **
# Boilerplate Code

# The Plugin API
The Go plugin API is fairly well documented here: http://godoc.org/github.com/uva-its/gopherbot/bot#Robot

The Bash and Ruby APIs aren't documented so well, however:
* Gopherbot includes several bash plugin examples and a fairly comprehensive `rubydemo` example
* The API libraries in `lib/` are fairly short and readble
* The Bash and Ruby APIs match the Go plugin API as closely as possible

### External Plugins
An external plugin is encapsulated in a single script, and added to the robot by adding a stanza to ExternalPlugins that gives the plugin name and path. The path can be absolute or relative; if relative, gopherbot checks Localdir then Installdir, so that a bot admin can modify a distributed plugin by simply copying it to `<localdir>/plugins` and editing it.

#### Structure
External plugins start with a bit of boilerplate code that loads the API library. This library is
responsible for translating function/method calls to http/json requests, and decoding the responses.

Once the library is loaded, ARGV[0] will contain a string command, optionally followed by arguments described below.

If the command is "configure", the plugin should dump it's default configuration to stdout; this is a yaml-formatted configuration that is common across all plugins. See `example.gopherbot/conf/plugins/example.yaml`.

If the command is "init", the plugin can perform any start-up actions, such as setting timers for automated messages.