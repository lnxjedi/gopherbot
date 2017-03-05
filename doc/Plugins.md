### Configuring plugins
The 'bot administrator can override default plugin configuration by putting a new `<pluginname>.yaml` file in conf/plugins subdirectory of the local config dir. Top-level settings in this file will override the default configuration for a plugin. Note that if you override **Help**, **CommandMatchers** or any other multi-part section, you should probably start by copying the entire section from the default config, as your settings will override the entire section.

See `example.gopherbot/conf/plugins/example.yaml` for a full example of a plugin configuration file.

## Writing your own plugins
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