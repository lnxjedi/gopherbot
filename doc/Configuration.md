Gopherbot is designed to be highly and flexibly configured, putting a lot of functionality in the hands of a Systems/DevOps engineer who may be deploying plugins written by a third party or someone else in the organization.

Table of Contents
=================

  * [Configuration Directories and Configuration File Precedence](#configuration-directories-and-configuration-file-precedence)
    * [Specifying Local Config](#specifying-local-config)
  * [Primary Configuration File \- gopherbot\.yaml](#primary-configuration-file---gopherbotyaml)
    * [Configuration Directives](#configuration-directives)
      * [AdminContact, Name and Alias](#admincontact-name-and-alias)
      * [Email and MailConfig](#email-and-mailconfig)
      * [Connection Protocol](#connection-protocol)
      * [Brain](#brain)
      * [AdminUsers and IgnoreUsers](#adminusers-and-ignoreusers)
      * [DefaultAuthorizer and DefaultElevator](#defaultauthorizer-and-defaultelevator)
      * [DefaultAllowDirect, DefaultChannels and JoinChannels](#defaultallowdirect-defaultchannels-and-joinchannels)
      * [ExternalPlugins](#externalplugins)
      * [LocalPort and LogLevel](#localport-and-loglevel)
  * [Plugin Configuration](#plugin-configuration)
    * [Plugin Configuration Directives](#plugin-configuration-directives)
      * [Disabled](#disabled)
      * [AllowDirect, DenyDirect, DirectOnly, Channels and AllChannels](#allowdirect-denydirect-directonly-channels-and-allchannels)
      * [CatchAll](#catchall)
      * [Users, RequireAdmin](#users-requireadmin)
      * [AuthorizedCommands, AuthorizeAllCommands, Authorizer and AuthRequire](#authorizedcommands-authorizeallcommands-authorizer-and-authrequire)
      * [TrustedPlugins](#trustedplugins)
      * [Elevator, ElevatedCommands and ElevateImmediateCommands](#elevator-elevatedcommands-and-elevateimmediatecommands)
      * [Help](#help)
      * [CommandMatchers, ReplyMatchers, and MessageMatchers](#commandmatchers-replymatchers-and-messagematchers)
      * [Config](#config)

# Configuration Directories and Configuration File Precedence

Gopherbot's configuration file loading is designed for automated deployments with most configuration stored in a git repository.

Credentials and other per-robot variables would normally be configured in the **install
directory**, and plugins, additional libraries, and common configuration would reside in an
optional **local config directory**, which is likely a git repository. A standard Linux
install with a configuration management tool would unzip the archive in `/opt/gopherbot` and
set e.g. the Slack Token in `/opt/gopherbot/conf/gopherbot.yaml`. Local configuration might
be in `/usr/local/etc/gopherbot`, and external plugins to load would be defined in
`/usr/local/etc/gopherbot/conf/gopherbot.yaml`. On Windows, the install directory would
normally be `C:\Program Files\Gopherbot`, and the configuration directory would be in
`C:\ProgramData\gopherbot`.

## Specifying Local Config

On startup, **Gopherbot** will search for a local configuration directory in the following order:
* A directory specified on the command line with `-c <dir>` or `--config <dir>`
* A platform-specific search path:
   * Linux/MacOS:
      * `/usr/local/etc/gopherbot`
      * `/etc/gopherbot`
      * `$HOME/.gopherbot`
   * Windows:
      * `C:\ProgramData\Gopherbot`
      * `%USERPROFILE%\.gopherbot` (`$env:USERPROFILE\.gopherbot`)

# Primary Configuration File - gopherbot.yaml

The robot's core configuration is obtained by simply loading `conf/gopherbot.yaml` from the **install directory** first, then the **local config directory** (if set), overwriting top-level items in the process.

## Configuration Directives
Note that some of this information is also available in the comments of the distributed `conf/gopherbot.yaml.sample`.

### AdminContact, Name and Alias

```yaml
AdminContact: Joe Admin, <sysadmin@my.org>
Name: Robbie Robot
Alias: ";"
```
`AdminContact` and `Name` are informational only, provided to users who ask for help. Note that you needn't specify `Name` for Slack - the robot will automatically obtain the name configured for the integration. The
alias can be used as shorthand for the robot's handle when addressing commands to the robot in a channel, and can be any of the following: `*+^$?\[]{}&!;:-%#@~<>/`

### Email and MailConfig

```yaml
Email: Robbie.Robot@my.org
MailConfig:
  Mailhost: smtp.my.org:25
  Authtype: plain # or 'none'
  User: robot
  Password: itsasecret
```
Both `Email` and `MailConfig` are required for the robot to be able to send email.

### Connection Protocol

Currently there are connector plugins for Slack and the terminal (for plugin development, see the `testcfg/` directory).

```yaml
Protocol: slack
ProtocolConfig:
  SlackToken: "xoxb-forgetitmackimnotputtingmytokenhere"
  MaxMessageSplit: 2
```
Whenever a plugin sends a very long message (longer than the
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

### AdminUsers and IgnoreUsers

```yaml
AdminUsers: [ 'alicek', 'bobj' ]
IgnoreUsers: [ 'danl', 'otherbot' ]
```
Users listed as admins have access to builtin administrative commands for viewing logs, changing log level,
reloading and terminating the robot. The robot will never respond to users listed in IgnoreUsers.

### DefaultAuthorizer and DefaultElevator

```yaml
DefaultAuthorizer: ldapauth
DefaultElevator: totp
```
Individual plugins may be configured to require command authorization or elevation (described below). In the absence of specific values for `Authorizer` and `Elevator`, plugins will use the defaults specified here to authorize or request elevation for specific commands. See the [Security Overview](Security-Overview.md) for a complete description of authorization and elevation, and the [Plugin Author's Guide](Plugin-Author's-Guide.md) for information on writing Authorization and Elevation plugins.

### DefaultAllowDirect, DefaultChannels and JoinChannels

```yaml
DefaultAllowDirect: true
DefaultChannels: [ 'general', 'random' ]
JoinChannels: [ 'security', 'infrastructure', 'lunch' ]
```
DefaultAllowDirect sets a robot-wide default value for AllowDirect, indicating whether a plugin's commands are accessible via direct message. DefaultChannels specify which channels a plugin will be active in if the plugin doesn't explicitly list it's channels. JoinChannels specify the channels the robot will try to join when logging in (though this isn't supported in the Slack connector).

### ExternalPlugins

```yaml
ExternalPlugins:
- Name: rubydemo
  Path: plugins/rubydemo
- Name: psdemo
  Path: plugins/psdemo.ps1
```
Most Gopherbot command plugins ship as single script files for any of several scripting languages. Installing
a new plugin only entails copying the plugin to an appropriate plugin directory (e.g. `<config dir>/plugins/`) and listing the plugin in the robot's `ExternalPlugins`, followed by a `reload` command.

### LocalPort and LogLevel

```yaml
LocalPort: 8880
LogLevel: info
```
Gopherbot command plugins communicate with the gopherbot process via JSON over http on a localhost port. The
port to use is configured with `LocalPort`. `LogLevel` specifies the initial logging level for the robot, one of `error`, `warn`, `info`, `debug`, or `trace`. The log level can also be adjusted on the fly by an administrator. Note that on Windows, debug and trace logging is only available in immediate mode during plugin development.

# Plugin Configuration

Gopherbot plugins are highly configurable with respect to visibility of plugins for various users and channels. In addition to providing a level of security, this can be very useful in large environments with many robots running many plugins, if only to keep the 'help' output to a minimum. The administrator can also configure **Authorization** and **Elevation** to further restrict sensitive commands. Additionally, help text and command routing is configured in yaml, allowing the administrator to e.g. provide synonyms for existing commands.

Plugins are configured by reading yaml from **three** locations, with top-level items from each successive read overriding the previous:
1. The default configuration is obtained from the plugin itself during plugin load, by calling the plugin with `configure` as the first argument; the robot looks for the plugin in the **local config directory** first (if present), then the **install directory**, getting the default configuration from the first found only.
2. Configuration is then loaded from `<install dir>/conf/plugins/<pluginname>.yaml`; this is where you might configure e.g. credentials required for a given plugin.
3. Finally, if a configuration directory is supplied, configuration is loaded from `<config dir>/conf/plugins/<pluginname>.yaml`; this is where you would likely configure a plugin's channels and security-related configuration (allowed users, authorization, etc.)

## Plugin Configuration Directives

### Disabled

```yaml
Disabled: true
```
Useful for disabling compiled-in Go plugins.

### AllowDirect, DenyDirect, DirectOnly, Channels and AllChannels

```yaml
AllowDirect: false  # default
DenyDirect: true    # default false, used when global DefaultAllowDirect = true
DirectOnly: true    # default false
```
```yaml
Channels:
- general
- devteam
```
```yaml
AllChannels: true
```
`AllowDirect` and `DenyDirect` interact with the global value of `DefaultAllowDirect` to determine if a plugin is available via direct message. Plugin-level directives override the global setting, and `DenyDirect` overrides `AllowDirect` (if both are mistakenly set). DirectOnly indicates the plugin is ONLY available by direct message (private chat). To specify the channels a plugin is available in, you can list the channels explicitly or set `AllChannels` to true. If neither is specified, the plugin falls back to the robot's configured `DefaultChannels`.

### CatchAll

```yaml
CatchAll: true  # default: false
```
If a plugin specifies `CatchAll`, and the robot receives a command that doesn't match a plugin, catchall plugins will be called with a command of `catchall`, and the message text as an argument. If configuring a catchall plugin, you should probably set `CatchAll: false` for the included `help` plugin.

### Users, RequireAdmin
```yaml
Users: [ 'alicek', 'bobc', 'bot:ServerWatch:*' ]
```
```yaml
RequireAdmin: true  # default: false
```
`Users` and `RequireAdmin` provide a simple mechanism to restrict visibility of a given plugin to a given list of users, or only bot administrators. If a given plugin isn't allowed for a given user, the robot will behave as if the plugin doesn't exist for that user, and won't provide help text. Note that `Users` can take globbing characters, mainly useful for matching messages from another bot or integration.

### AuthorizedCommands, AuthorizeAllCommands, Authorizer and AuthRequire

```yaml
Authorizer: ldapcheck      # overrides the global DefaultAuthorizer if specified
AuthRequire: serveradmins    # not required for using authorization
AuthorizeAllCommands: true
# - or -
AuthorizedCommands:
- createserver
- destroyserver
```
Authorization support lets a plugin delegate command authorization determinations to another plugin, which may be shared among a family of related plugins. Note that if authorization fails, the user is notified and the target plugin is never called. The optional `AuthRequire` allows the target plugin to specify a group or role name the user should be authorized against. See the [Plugin Author's Guide](Plugin-Author's-Guide.md) for a full description of Authorizer plugins.

### TrustedPlugins

```yaml
TrustedPlugins:
- server-manager
- dns-manager
```
`TrustedPlugins` determines which plugins are allowed to use the `CallPlugin(...)` method to call this plugin. When called via `CallPlugin`, there is no authorization or elevation check performed for the target; rather, the target _trusts_ that the calling plugin configured appropriate authorization and/or elevation.

### Elevator, ElevatedCommands and ElevateImmediateCommands

```yaml
Elevator: duo # overrides the global DefaultElevator if specified
ElevatedCommands: [ "startserver", "alias" ]
ElevateImmediateCommands:
- terminate
- destroyvolume
- disablesite
```
The `Elevate*` directives specify that certain commands in a plugin require additional identity verification to run. Gopherbot ships with two plugins for this functionality, with Google Authenticator style TOTP being the most common. `ElevatedCommands` are subject to a timeout during which additional verification won't be required;
`ElevateImmediate` commands always prompt for additional verification. Additionally, individual commands can use
the `Elevate(bool: immediate)` method to require elevation based on conditional logic in the command, or for all commands in the unusual case of requiring elevation for all commands in a plugin.

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
- Command: stopserver
  Regex: '(?i:stop server(?: ([\w-.]+))?'
  Contexts: [ "server" ]
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

(*EXPERIMENTAL*) "Contexts" tells the robot what kind of thing a capture group corresponds to. When the robot
sees that e.g. a capture group corresponds to the context "server", it stores that in short-term memory (~7
minutes). Then, if the user doesn't supply a corresponding capture group in a subsequent command, or supplies "it", the robot will check it's short term memory to see if it knows which "server" you're talking about. Short
term memories are per-user/channel combination, but not per plugin; so if two plugins specify `Contexts` of, say, `server` and `deployment`, the short term memory of which server or deployment being referred to is available to both. If the memory doesn't exist or has expired from short-term memory, the robot will simply
reply that it doesn't know what "server" or "deployment" you're talking about.

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
method. Examples of this can be seen in the included `plugins/rubydemo.rb`, `plugins/weather.rb`, `plugins/psdemo.ps1`, and `goplugins/knock/*`. This allows, for instance, configuring additional knock-knock jokes without modifying or
recompiling the plugin, subject to the caveat that modifying the configuration means copying the entire
`Config:` section to `conf/plugins/<plugginname>.yaml`.
