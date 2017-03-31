Gopherbot is designed to be highly and flexibly configured, putting a lot of functionality in the hands of
a system administrator who may be deploying plugins written by a third party or someone else in the organization.

Table of Contents
=================

  * [Table of Contents](#table-of-contents)
  * [Configuration Directories and Configuration File Precedence](#configuration-directories-and-configuration-file-precedence)
  * [Primary Configuration File \- gopherbot\.yaml](#primary-configuration-file---gopherbotyaml)
    * [Configuration Directives](#configuration-directives)
      * [AdminContact, Name and Alias](#admincontact-name-and-alias)
      * [Email and MailConfig](#email-and-mailconfig)
      * [Connection Protocol](#connection-protocol)
      * [Brain](#brain)
      * [AdminUsers, IgnoreUsers, and Elevation](#adminusers-ignoreusers-and-elevation)
      * [DefaultChannels and JoinChannels](#defaultchannels-and-joinchannels)
      * [ExternalPlugins](#externalplugins)
      * [LocalPort and LogLevel](#localport-and-loglevel)
  * [Plugin Configuration](#plugin-configuration)
    * [Plugin Configuration Directives](#plugin-configuration-directives)
      * [Disabled](#disabled)
      * [DisallowDirect, DirectOnly, Channels and AllChannels](#disallowdirect-directonly-channels-and-allchannels)
      * [CatchAll](#catchall)
      * [Users, RequireAdmin](#users-requireadmin)
      * [ElevatedCommands and ElevateImmediateCommands](#elevatedcommands-and-elevateimmediatecommands)
      * [Help](#help)
      * [CommandMatchers, ReplyMatchers, and MessageMatchers](#commandmatchers-replymatchers-and-messagematchers)
      * [Config](#config)

# Configuration Directories and Configuration File Precedence

Gopherbot's configuration file loading is designed around installation where credentials and other per-robot variables are configured in the **install directory**, and plugins, additional libraries, and common configuration reside in a **local config directory**, which is likely a git repository. A standard Linux install with a configuration management tool would unzip the archive in `/opt/gopherbot` and set e.g. the Slack Token in `/opt/gopherbot/conf/gopherbot.yaml`. Local configuration might be in `/usr/local/etc/gopherbot`, and external
plugins to load would be defined in `/usr/local/etc/gopherbot/conf/gopherbot.yaml`. On Windows, the install directory would normally be `C:\Program Files\Gopherbot`, and the configuration directory would be in `C:\Windows\gopherbot`.

# Primary Configuration File - gopherbot.yaml
The robot's core configuration is obtained by simply loading `conf/gopherbot.yaml` from the **install directory** first, then the **local config directory**, overwriting top-level items in the process.

## Configuration Directives
Note that some of this information is also available in the comments of the distributed `conf/gopherbot.yaml`.

### AdminContact, Name and Alias
```yaml
AdminContact: Joe Admin, <sysadmin@my.org>
Name: Robbie Robot
Alias: ";"
```
Admin contact and Name are informational only, provided to users who ask for help. Note that you needn't specify `Name` for Slack - the robot will automatically obtain the name configured for the integration. The
alias can be used as shorthand for the robot's handle when addressing commands to the robot in a channel.

### Email and MailConfig
```yaml
Email: Robbie.Robot@my.org
MailConfig:
  Mailhost: smtp.my.org:25
  Authtype: plain # or 'none'
  User: robot
  Password: itsasecret
```
Email is informational, but required for the robot to be able to send email.

### Connection Protocol
```yaml
Protocol: slack
ProtocolConfig:
  SlackToken: "xoxb-forgetitmackimnotputtingmytokenhere"
  MaxMessageSplit: 2
```
Currently the only connector plugin is for Slack. Whenever a plugin sends a very long message (longer than the
Slack maximum message length), the slack connector will automatically break the message up into shorter
messages; MaxMessageSplit determines the maximum number to split a message into before truncating.

### Brain
```yaml
Brain: file
BrainConfig:
  BrainDirectory: brain
```
Gopherbot ships with a simple file-based brain, with pluggable support for creating e.g. a redis based brain.
The BrainDirectory can be given as an absolute path or as a sub-directory of the local config directory.

### AdminUsers, IgnoreUsers, and Elevation
```yaml
AdminUsers: [ 'alicek', 'bobj' ]
IgnoreUsers: [ 'danl', 'otherbot' ]
ElevateMethod: topt
ElevateConfig:
  TimeoutSeconds: 3600
  TimeoutType: idle
```
Users listed as admins have access to builtin administrative commands for viewing logs, changing log level,
reloading and terminating the robot. The robot will never respond to users listed in IgnoreUsers.

Individual commands can be configured to require elevation, normally requiring the user to provide some variety of MFA token before completing the command. More information on
elevation is in [Configuring Elevation](Elevation.md).

### DefaultChannels and JoinChannels
```yaml
DefaultChannels: [ 'general', 'random' ]
JoinChannels: [ 'security', 'infrastructure', 'lunch' ]
```
DefaultChannels specify which channels a plugin will be active in if the plugin doesn't explicitly list it's
channels. JoinChannels specify the channels the robot will try to join when logging in (though this isn't
supported in the Slack connector).

### ExternalPlugins
```yaml
ExternalPlugins:
- Name: rubydemo
  Path: plugins/rubydemo
- Name: psdemo
  Path: plugins/psdemo.ps1
```
Most Gopherbot command plugins ship as single script files for any of several scripting languages. Installing
a new plugin only entails copying the plugin to an appropriate plugin directory (e.g. `<config dir>/plugins/`) and
listing the plugin in the robot's `ExternalPlugins`.

### LocalPort and LogLevel
```yaml
LocalPort: 8880
LogLevel: info
```
Gopherbot command plugins communicate with the gopherbot process via JSON over http on a localhost port. The
port to use is configured with LocalPort. LogLevel specifies the initial logging level for the robot, one of `error`, `warn`, `info`, `debug`, or `trace`. The log level can also be adjusted on the fly by an administrator. Note that on Windows, debug and trace logging is only available in immediate mode during plugin development.

# Plugin Configuration
Gopherbot plugins are highly configurable with respect to which users can access a plugin, and in which channels. This can be very useful in large environments with many robots running many plugins, if only to keep
the 'help' output to a minimum. Additionally, help text and command routing is configured in yaml, allowing
the user to e.g. provide synonyms for existing commands.

Plugins are configured by reading yaml from **three** locations, with top-level items from each successive read
overwriting the previous:
1. The default configuration is obtained from the plugin itself during plugin load, by calling the plugin with `configure` as the first argument
2. Configuration is then loaded from `<install dir>/conf/plugins/<pluginname>.yaml`; this is where you might configure e.g. credentials required for a given plugin
3. Finally configuration is loaded from `<config dir>/conf/plugins/<pluginname>.yaml`; this is where you would likely configure a plugin's channels and security-related configuration (allowed users, etc.)

## Plugin Configuration Directives
### Disabled
```yaml
Disabled: true
```
Useful for disabling compiled-in Go plugins.

### DisallowDirect, DirectOnly, Channels and AllChannels
```yaml
DisallowDirect: false # default
DirectOnly: true      # default: false
```
```yaml
Channels:
- general
- devteam
```
```yaml
AllChannels: true
```
DisallowDirect and DirectOnly determine the availability of a given plugin by direct messaging the robot
(private chat). To specify the channels a plugin is available in, you can list the channels explicitly or
set `AllChannels` to true. If neither is specified, the plugin falls back to the robot's configured
`DefaultChannels`.

### CatchAll
```yaml
CatchAll: true  # default: false
```
If a plugin specifies `CatchAll`, it will be called with a command of `catchall`, and the message text as an
argument.

### Users, RequireAdmin
```yaml
Users: [ 'alicek', 'bobc' ]
```
```yaml
RequireAdmin: true  # default: false
```
Users and RequireAdmin can be used to restrict a given plugin to a given list of users, or only bot
administrators. If a given plugin isn't allowed for a given user, the robot will behave as if the plugin
doesn't exist for that user, and won't provide help text.

### ElevatedCommands and ElevateImmediateCommands
```yaml
ElevatedCommands: [ "startserver", "alias" ]
ElevateImmediateCommands:
- terminate
- destroyvolume
- disablesite
```
The Elevate* directives specify that certain commands in a plugin require additional verification to run.
Gopherbot ships with two plugins for this functionality, with Google Authenticator style TOTP being the most
common. ElevatedCommands are subject to a timeout during which additional verification won't be required;
ElevateImmediate commands always prompt for additional verification. Additionally, individual commands can use
the `Elevate(bool: immediate)` method to require elevation based on conditional logic in the command.

### Help
```yaml
Help:
- Keywords: [ "hosts", "lookup", "dig", "nslookup" ]
  Helptext: [ "(bot), dig <hostname|ip>" ]
```
Gopherbot ships with a simple keyword based help system; when the user requests `help <keyword>`, the robot
will list the example commands in `Helptext` for the given keyword. The string `(bot)` will be
replaced by the robot's handle.

Note that if you wish to configure additional help, you'll need to copy the entire `Help` section from the
plugin's default configuration to the appropriate `<pluginname>.yaml` file.

### CommandMatchers, ReplyMatchers, and MessageMatchers
```yaml
CommandMatchers:
- Command: hosts
  Regex: '(?i:hosts?|lookup|dig|nslookup) ([\w-. ]+)'
- Command: hostname
  Regex: '(?i:hostname)'
MessageMatchers:
- Command: chuck
  Regex: '(?i:Chuck Norris)'
ReplyMatchers:
- Label: phone
  Regex: '\(\d\d\d\)\d\d\d-\d\d\d\d'
```
The `*Matchers` directives specify Go language regular expressions for matching messages. `CommandMatchers` are
checked whenever a user directs a message to the robot directly, using it's handle or alias. `MessageMatchers`
are checked against all messages seen in the channels for which the plugin is active; this is useful for e.g.
a **Chuck Norris** type plugin. Whenever a command or message is matched, the external plugin is called with the
first argument being the given `Command`, and subsequent arguments corresponding to matching groups in the
regular expression.

`ReplyMatchers` are used whenever the plugin uses the `WaitForReply` method for interactive plugins like the
built-in knock-knock joke plugin; the first argument to `WaitForReply` specifies the label of the regex to wait
for.

Note that for each of these, the robot internally modifies the regex to be insensitive to white space; this was
frequently a problem when users were cutting and pasting arguments to robot commands.

### Config
```yaml
Config:
  Replies:
  - "Consider yourself rubied"
  - "Waaaaaait a second... what do you mean by that?"
  - "I'll ruby you, but not right now - I'll wait 'til you're least expecting it..."
  - "Crap, sorry - all out of rubies"
```
The `Config` directive provides a section of free-form yaml that the plugin can access via the `GetPluginConfig`
method. Examples of this can be seen in the included `plugins/rubydemo`, `plugins/psdemo.ps1`, and `goplugins/knock/*`. This allows, for instance, configuring additional knock-knock jokes without modifying or
recompiling the plugin, subject to the caveat that modifying the configuration means copying the entire
`Config:` section to `conf/plugins/<plugginname>.yaml`.
