Gopherbot is designed to be highly and flexibly configured, putting a lot of functionality in the hands of a Systems/DevOps engineer who may be deploying plugins written by a third party or someone else in the organization.

Table of Contents
=================

  * [Configuration Directories and Configuration File Precedence](#configuration-directories-and-configuration-file-precedence)
    * [Specifying Config](#specifying-config)
  * [Primary Configuration File \- gopherbot\.yaml](#primary-configuration-file---gopherbotyaml)
    * [Configuration Directives](#configuration-directives)
      * [AdminContact, Name and Alias](#admincontact-name-and-alias)
      * [Email and MailConfig](#email-and-mailconfig)
      * [Connection Protocol](#connection-protocol)
      * [DefaultMessageFormat](#defaultmessageformat)
      * [Brain](#brain)
      * [AdminUsers and IgnoreUsers](#adminusers-and-ignoreusers)
      * [DefaultAuthorizer and DefaultElevator](#defaultauthorizer-and-defaultelevator)
      * [DefaultAllowDirect, DefaultChannels and JoinChannels](#defaultallowdirect-defaultchannels-and-joinchannels)
      * [ExternalScripts](#externalscripts)
      * [LocalPort and LogLevel](#localport-and-loglevel)
  * [Task Configuration](#task-configuration)
    * [Plugins and Jobs](#plugins-and-jobs)
    * [Task Configuration Directives](#plugin-configuration-directives)
      * [Disabled](#disabled)
      * [AllowDirect, DirectOnly, Channels and AllChannels](#allowdirect-directonly-channels-and-allchannels)
      * [CatchAll](#catchall)
      * [Users, RequireAdmin, AdminCommands](#users-requireadmin-admincommands)
      * [AuthorizedCommands, AuthorizeAllCommands, Authorizer and AuthRequire](#authorizedcommands-authorizeallcommands-authorizer-and-authrequire)
      * [Elevator, ElevatedCommands and ElevateImmediateCommands](#elevator-elevatedcommands-and-elevateimmediatecommands)
      * [Help](#help)
      * [NameSpace](#namespace)
      * [CommandMatchers, ReplyMatchers, and MessageMatchers](#commandmatchers-replymatchers-and-messagematchers)
      * [Config](#config)

# Configuration Directories and Configuration File Precedence

Gopherbot's configuration file loading is designed for automated deployments with most configuration stored in a git repository.

Credentials and other per-robot variables would normally be configured in the **install
directory**, and plugins, additional libraries, and common configuration would reside in an
optional **config directory**, which is likely a git repository. A standard Linux
install with a configuration management tool would unzip the archive in `/opt/gopherbot` and
set e.g. the Slack Token in `/opt/gopherbot/conf/robot.yaml`. Configuration might
be in `/usr/local/etc/gopherbot`, and external plugins to load would be defined in
`/usr/local/etc/gopherbot/conf/robot.yaml`. On Windows, the install directory would
normally be `C:\Program Files\Gopherbot`, and the configuration directory would be in
`C:\ProgramData\Gopherbot`.

## Specifying Config

On startup, **Gopherbot** will search for a configuration directory in the following order:
* A directory specified on the command line with `-c <dir>` or `--config <dir>`
* A platform-specific search path:
   * Linux/MacOS:
      * `/usr/local/etc/gopherbot`
      * `/etc/gopherbot`
      * `$HOME/.gopherbot`
   * Windows:
      * `C:\ProgramData\Gopherbot`
      * `%USERPROFILE%\.gopherbot` (`$env:USERPROFILE\.gopherbot`)

# Primary Configuration File - robot.yaml

The robot's core configuration is obtained by simply loading `conf/robot.yaml` from the **install directory** first, then the **config directory** (if set), overwriting top-level items in the process.

## Configuration Directives
Note that some of this information is also available in the comments of the distributed `conf/robot.yaml.sample`.

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

Currently there are connector plugins for Slack, the terminal, and a special test
connector for automated integration testing. For sample configurations for these
connectors, see the `cfg/` directory.

```yaml
Protocol: slack
ProtocolConfig:
  SlackToken: "xoxb-forgetitmackimnotputtingmytokenhere"
  MaxMessageSplit: 2
```
Whenever a plugin sends a very long message (longer than the
Slack maximum message length), the slack connector will automatically break the message up into shorter
messages; MaxMessageSplit determines the maximum number to split a message into before truncating.

### DefaultMessageFormat

```yaml
DefaultMessageFormat: Variable # default: Raw
```

The `DefaultMessageFormat` determines how the robot will format it's messages to the users when formatting
isn't explicit. The default value of `Raw` means send the message unmodified to the chat service; this will
result in protocol-specific formatting. For instance, slack will display `_italics_` as "_italics_" without displaying the
underscores. Setting `DefaultMessageFormat: Variable` would instead display "\_italics\_". This setting is mainly useful
for plugins written to work across multiple chat platforms.

### Encryption
For CI/CD applications where the brain may be used for storing secrets, **Gopherbot** requires an encryption key to use with AES-GCM for encrypting and decrypting memories sent to the storage engine.

**Gopherbot** uses the provided key as a temporary key to decrypt the '*real*' key which is randomly generated and stored in a memory region protected by guard pages (github.com/awnumar/memguard). The most secure method of providing the key is with the built-in administrator command `initialize encryption <key>`, only allowed as a direct message.

*NOTE*: The author holds no security or encryption certifications or credentials, and this functionality is provided with no guarantees against sophisticated attacks. Any security sensitive use for protecting high-value assets should start with a thorough security audit of the code. (PRs welcome) That being said, if the brain is initialized interactively, it should be quite difficult for another process running as root or the robot's UID to obtain the key and decrypt memories.

The key can also be provided in `robot.yaml`, or in the `GOPHER_ENCRYPTION_KEY` environment variable, referenced in the default `robot.yaml`.
```yaml
EncryptionKey: ThisNeedsToBeAStringLongerThan32bytesForItToWork # optional
```

### Brain

**Gopherbot** supports a simple key-value brain with multiple storage backends for storing long-term memories as JSON blobs that are serialized/de-serialized in to simple data structures.

#### Full-brain Encryption
Specifying `EncryptBrain: true` will turn on encryption for **ALL** stored memories, not just stored secrets.

#### File Brain

```yaml
Brain: file
BrainConfig:
  BrainDirectory: brain
```
The file brain stores the robots memories in files in the local filesystem.
The BrainDirectory can be given as an absolute path or as a sub-directory of the config directory (or install
directory if no config directory is set).

#### DynamoDB Brain

```yaml
Brain: dynamo
BrainConfig:
  TableName: MyBot
  Region: "us-east-1"
  # You can leave these blank and set AWS_ACCESS_KEY_ID
  # and AWS_SECRET_ACCESS_KEY in environment variables
  AccessKeyID: ""
  SecretAccessKey: ""
```
The DynamoDB brain provides durable long-term storage in the AWS cloud. You'll
need to create a table and API credentials for accessing the table. If `AccessKeyID`
and `SecretAccessKey` aren't provided, the AWS Go library will use standard means
of obtaining credentials.

The table needs a Primary Key `Memory`, type `String`.

The minimum policy required for your robot to use e.g. the `MyBot` table is:
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "AllowBot",
            "Effect": "Allow",
            "Action": [
                "dynamodb:PutItem",
                "dynamodb:DescribeTable",
                "dynamodb:GetItem"
            ],
            "Resource": "arn:aws:dynamodb:*:*:table/MyBot"
        }
    ]
}
```

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
DefaultAllowDirect: true # default
DefaultChannels: [ 'general', 'random' ]
JoinChannels: [ 'security', 'infrastructure', 'lunch' ]
```
DefaultAllowDirect sets a robot-wide default value for AllowDirect, indicating whether a plugin's commands are accessible via direct message; `true` if not otherwise specified. DefaultChannels specify which channels a plugin will be active in if the plugin doesn't explicitly list it's channels. JoinChannels specify the channels the robot will try to join when logging in (though this isn't supported in the Slack connector).

### ExternalScripts

```yaml
ExternalScripts:
- Name: rubydemo
  Path: plugins/rubydemo
- Name: psdemo
  Path: plugins/psdemo.ps1
```
Most Gopherbot command plugins ship as single script files for any of several scripting languages. Installing
a new plugin only entails copying the plugin to an appropriate plugin directory (e.g. `<config dir>/plugins/`) and listing the plugin in the robot's `ExternalScripts`, followed by a `reload` command.

### LocalPort and LogLevel

```yaml
LocalPort: 8880
LogLevel: info
```
Gopherbot external scripts communicate with the gopherbot process via JSON over http on a localhost port. The port to use is configured with `LocalPort`; `0` will automatically allocate an available port. `LogLevel` specifies the initial logging level for the robot, one of `error`, `warn`, `info`, `debug`, or `trace`. The log level can also be adjusted on the fly by an administrator.

# Task Configuration

Gopherbot tasks (jobs and plugins) are highly configurable with respect to visibility in channels, security, and input arguments and parameters.

Task are configured by reading yaml from up to **three** locations, with top-level items from each successive read overriding the previous:
1. For command plugins, the default configuration is obtained from the plugin itself during plugin load, by calling the plugin with `configure` as the first argument; the robot looks for the plugin in the **config directory** first (if present), then the **install directory**, getting the default configuration from the first found only.
2. Configuration is then loaded from `<install dir>/conf/<jobs|plugins>/<taskname>.yaml`; this is where you might configure e.g. credentials required for a given task.
3. Finally, if a configuration directory is supplied, configuration is loaded from `<config dir>/conf/<jobs|plugins>/<taskname>.yaml`; this is where you would likely store non-sensitive configuration directive to be stored in a **git** repository.

## Plugins and Jobs

Gopherbot supports two types of tasks; `plugins` and `jobs`.

## Task Configuration Directives

### Disabled

```yaml
Disabled: true
```
Useful for disabling compiled-in Go plugins.

### AllowDirect, DirectOnly, Channels and AllChannels

```yaml
AllowDirect: false  # only needed for overriding global DefaultAllowDirect
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
`AllowDirect` determines if a plugin is available via direct message, and is only needed to override the global value for `DefaultAllowDirect`. DirectOnly indicates the plugin is ONLY available by direct message (private chat), normally for security-sensitive commands. To specify the channels a plugin is available in, you can list the channels explicitly or set `AllChannels` to true. If neither is specified, the plugin falls back to the robot's configured `DefaultChannels`.

### CatchAll

```yaml
CatchAll: true  # default: false
```
If a plugin specifies `CatchAll`, and the robot receives a command that doesn't match a plugin, catchall plugins will be called with a command of `catchall`, and the message text as an argument. If configuring a catchall plugin, you should probably set `CatchAll: false` for the included `help` plugin.

### Users, RequireAdmin, AdminCommands
```yaml
Users: [ 'alicek', 'bobc', 'bot:ServerWatch:*' ]
```
```yaml
RequireAdmin: true  # default: false
```
```yaml
AdminCommands:
- list
```
`Users` and `RequireAdmin` provide a simple mechanism to restrict visibility of a given plugin to a given list of users, or only bot administrators. If a given plugin isn't allowed for a given user, the robot will behave as if the plugin doesn't exist for that user, and won't provide help text. Note that `Users` can take globbing characters, mainly useful for matching messages from another bot or integration.

`AdminCommands` restricts certain commands to bot administrators only. The help for the command will still be visible, so the text should probably include e.g. `(bot administrators only)`.

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

### NameSpace
By default, **Gopherbot** separates memories and parameters by task (task/job/plugin) name. In some cases, tasks may need to share parameters and memories. To do so, a given plugin, task or job can have the "NameSpace" attribute, referring to another plugin, task or job, with the follow effects:
* For memories, the given namespace is used
* For parameters, parameters from the given namespace are merged with current task parameters, with locally defined parameters taking precedence

```yaml
NameSpace: builds
```

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
checked whenever a user directs a message to the robot directly, using it's handle or alias; start- and end-of-line anchors
(`^` & `$`) are automatically added, and should not be in the regex. `MessageMatchers` are not automatically anchored, and
are checked against all messages seen in the channels for which the plugin is active; this is useful for e.g.
a **Chuck Norris** type plugin, where any mention of The Great One should generate a response. Whenever a command or message is matched, the external plugin is called with the
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
The `Config` directive provides a section of free-form yaml that the plugin can access via the `GetTaskConfig`
method. Examples of this can be seen in the included `plugins/rubydemo.rb`, `plugins/weather.rb`, `plugins/psdemo.ps1`, and `goplugins/knock/*`. This allows, for instance, configuring additional knock-knock jokes without modifying or
recompiling the plugin, subject to the caveat that modifying the configuration means copying the entire
`Config:` section to `conf/plugins/<plugginname>.yaml`.
