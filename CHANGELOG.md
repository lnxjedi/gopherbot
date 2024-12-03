# v2.16.0 - Initial Support for Julia Extensions
I knew that someday I would be able to take the existing support for Python and/or Ruby, feed it to an AI, and get a working Julia library. In short, check `plugins/samples/echo.jl` for the first mimimal working Julia plugin. Notes:
* Julia, weighing in at a whopping 1G, hasn't been integrated into any prebuilt containers
* If you want to do so yourself:
  * Install a julia glibc tarball in e.g. `/opt/julia` and symlink `/usr/bin/julia` to `/opt/julia/bin/julia`
  * In your container build, the robot user needs to: `using Pkg; Pkg.add("JSON"); Pkg.add("HTTP"); exit()` to install the required JSON and HTTP libraries to `$HOME/.julia`

As with Ruby and Python, if your robot includes `lib/SomethingCool.jl`, your scripts should be able to `using SomethingCool` to load it as a library.

# v2.15.5 - Updated PrivSep
It appears that Gopherbot's ability to drop privileges to nobody stopped working some time ago. This version updates privsep.go to use `runtime.LockOSThread()` (as before) with `syscall.Sysscall(SYS_SETREUID, ...)` to drop privileges when running external scripts. See the [documentation](https://lnxjedi.github.io/gopherbot/install/ManualInstall.html#privilege-separation) for more information about privilege separation.

# v2.15.2 - Security Updates
Mainly this updates a Go dependency. Note that "CodeQL" still has a few issues related to the external script API that could be exploitable by untrusted extensions. My recommendation on that is and always will be: "don't run untrusted extensions on important robots". Maybe someday I'll audit the code and see if there's some means of making it safer (though likely never completely safe) to run untrusted plugins. Given the very slight resource requirements of a Gopherbot robot, my official recommendation would be:
* If you want silly third-party plugins, run them in a separate robot that doesn't have access to anything important
* Remove that robot's "manage" key (the one with read-write git permissions)

# v2.15.1 - Bug Fixes
* Fixes the return values in the script libraries to match the new values in Go
* Fixes the heuristic in the Slack connector for when to send an ephemeral message
* Fixes git pull and switch-branch to ensure the correct branch is pulled

# v2.15.0 - Dynamic Go Extensions

The biggest, coolest, most exciting news for v2.15 is the ability to write new tasks, jobs and plugins in Go that are loaded and interpreted from the filesystem - thus, you can now add Go extensions to your robot's repository. To demonstrate this, some of Gopherbot's core and included tasks were modified to run this way. See:
* jobs/go-*
* plugins/go-*
* tasks/go-*

The differences between compiled-in and dynamic go extensions (other than the obvious) are fairly minimal, code-wise. For comparison, examine the built-in Go tasks in `gotasks/`, `goplugins/` and `gojobs/` - these are added individually in `/modules.go`, whereas dynamic extensions, like other external scripts, are defined in `robot.yaml`.

## Unlikely breaking change - update to Go `GetTaskConfig()`
For better support of dynamic Go modules, and to bring the Go library more inline with the libraries for Ruby and Python, the Go `GetTaskConfig()` method was modified to simply unmarshal JSON into a struct pointer. New return values were also added. See the definitions under `robot/` for more information.

## `cbot` script completion
If you add (or symlink) the `cbot` script somewhere in your `$PATH`, you can add this line to your `.bashrc` for tab completion:
```
source <(cbot completion)
```
It's a little frivolous, but I've come to expect tab completion for virtually every command I use. Thanks to AI, it's amazingly ... complete.

## Support for better local dev security
To keep from storing unencrypted values for `GOPHER_ENCRYPTION_KEY` and `GOPHER_DEPLOY_KEY` in the `<robot>.env` file, you can remove these lines and instead:
* Start your dev container as usual
* `code $HOME/.env` and add the encryption key and deploy key, likely from a password manager secure note
* Remove the container when finished to remove the file

# v2.14.1 - Fixes for restore, update
Haha! This isn't the first time I've made a minor release with significant breakage, alas. So, I fired up Clu - "Okay, Clu, tonight we check everything in the right-hand column."

So, as of now, Clu seems able to save and restore his memories, and the update job works. Since my production robots are having this problem, I'll fall back on the meme: "I don't always test my code, but when I do, I test in production."

# v2.14.0 - New "validate" command
The primary goal of this release was adding a new CLI "validate" command to check the configuration files for a repository. For instance, one of my most common typos:
```
[clu:~]$ gopherbot validate clu
Validating configuration
Error: Validating configured/custom configuration: validation error in 'clu/conf/robot.yaml': yaml: unmarshal errors:
  line 72: field Paramters not found in type bot.TaskSettings
Fatal: Loading initial configuration: Loading configuration file: validation error in 'clu/conf/robot.yaml': yaml: unmarshal errors:
  line 72: field Paramters not found in type bot.TaskSettings
# After fixing ...
[clu:~]$ gopherbot validate clu
Validating configuration
Warning: Plugin 'groups' has custom config, but none is configured
Info: Initialized brain provider 'mem'
Configuration valid
```

The goal here, of course, is to allow robot administrators to check their robots before (re-)deploying them, to ensure they will start since configuration errors are fatal when the robot first starts. Note that updating the robot with invalid configuration will just throw an error on reload and leave the original configuration in place.

## New "ssh-agent" pipeline task
Gopherbot now includes code for basic ssh-agent functionality for use in pipelines. See `gotasks/ssh-agent`. The goal for v3.0.0 is to eliminate external dependencies for bootstrapping - this is the first step.

## Potentially Breaking Changes
Another goal for Gopherbot v3 is to clean up some of the code, and remove ugly cruft that makes development more painful with little gain - this means removing functionality I haven't used in years, and definitely don't test/think about when releasing new versions.

### Removed 'daemonize'
None of the modern deployment methods for a Gopherbot robot require gopherbot to daemonize itself, so that code has been removed. Instead, a process supervisor like systemd generally handles running gopherbot as a system service.

### Fixed location for 'custom'
Of all the robots I've deployed, I've never specified a path to a configuration directory, always letting it default to "custom". If a system needs to run more than one robot, you need to ensure they start with a unique initial working directory where the process can create/read/write the "custom/" directory.

# v2.13.0 - Configuration file validation
The major change here is something I've wanted for ages - configuration files should now be properly validated, preventing the kind of "typo misconfig" that I've dealt with so often.

## GopherCI - Unlikely breaking change
At one time I thought Gopherbot would be more Jenkins-like in building software - but, honestly, it was never very good at it where so many other projects are. Now Gopherbot focuses more on doing what it does best - secure DevOps automation in team chat. So long, GopherCI. (Note: I'm pretty sure I'm the only person who ever used this, and didn't much care for it)

# v2.12.0 - Slack hidden command reflection
While I completly love the new ability for Gopherbot Slack bots to respond to "hidden" (slash "/") commands, it has always bothered me that I can't see the command I just sent. This version of Gopherbot "reflects" those messages by default; to disable, add `DisableReflection: true` to your Slack protocol configuration.

One other small issue, I can't believe it's taken so many years to see - keyword help is now case-insensitive, since `help GitHub` should match all commands that have "github" as a keyword.

# v2.11.1 - Improved Slack "raw" Message Splitting
On occasion the AI plugin could generate very long Slack messages containing one or more code blocks. The old message splitting algorithm just blindly split up the message and sent chunks, breaking on newlines when possible. The new algorithm now keeps track of code blocks, and closes and re-opens them between messages.

# v2.11.0 - Hidden Commands and Support for Slack Slash Commands
When Gopherbot's Slack connector migrated from RTM to socketmode events, I added bare minimum support for "slash commands" - if you set up your robot with e.g. "/clu" for a slash command, you could type "/clu ping" and it would dutifully respond "@parsley PONG" in the channel - but nobody would have seen the original ping command, since slash commands are hidden. Version 2.11.0 adds a more useful functionality for slash commands called "hidden commands", meaning the user can message the robot and receive a reply that is hidden from other users. The interfaces are defined in a generic way so that future connectors could also support this functionality in a connector-specific way, but for now I will only document how it's implemented in Slack.

## AllowedHiddenCommands - potentially breaking change
Where before every plugin command would respond to e.g. "/clu do something" the same as "clu, do something", now each plugin must define a list of commands that are allowed to be hidden with an `AllowedHiddenCommands:` array in the plugin yaml configuration. Since Gopherbot allows an adminstrator to pin certain commands to certain channels, and visibility of commands is one layer of security, this feature returns some control back to the adminstrator.

## New GOPHER_HIDDEN_COMMAND Environment Variable
When a plugin runs from a hidden command, the engine will set `GOPHER_HIDDEN_COMMAND=true`; otherwise it's unset.

## Slack Implementation with Ephemeral Messages
The nicest feature of this new functionality is how it's implemented in Slack. Simply, when a command is issued as a slash command, all replies are sent as an "ephemeral message" to the user:
* Slack tags these messages as "Only visible to you"
* They are not kept in team history
* They do not persist between sessions / reloads
That is all to say, Slack hidden messages behave much like many other Slack integrations you've used, and don't clutter up channels with output that likely isn't useful to others.

## Updates to built-in help
Absolutely the most useful application of this new feature is the update to the built-in help system:
* Built-in help commands are listed as `AllowedHiddenCommands` by default, so in any given channel a user can type e.g. `/clu help` and receive contextual help for the current channel in the same pane, without spamming everybody else in the channel, or creating a thread.
* Expansion of helplines now accepts a prefix before "(bot)", so you can now add a helptext line of the form "/(bot) do something - do anything at all", and it will render as e.g. "`/Clu do something` - do anything at all".
* Protocol connectors now define a `DefaultHelp()` method that can return a non-zero array of help lines that override the engine defaults when the help command is issued without a keyword. Thus, a Slack robot will add e.g. "`/Clu help <keyword>` - get help for the provided \<keyword\>".

## (Development Notes) Additional Changes to the Core
To support this functionality, the message sending methods were updated. Before, whenever a connector had an incoming message, it was packaged up in to a `*robot.ConnectorMessage`, with a `MessageObject` field containing a protocol-specific `interface{}` corresponding to an object received by the protocol (e.g. `*slack.Event`), and a similar `Client interface{}` corresponding to a client object for the protocol (e.g. `*slack.Client`). Now this same object is passed back as an argument in the message sending functions, allowing the connector to make context-aware decisions on how to handle message sends. In the case of Slack, it's used to determine that an ephemeral message should be sent to the user. At the same time, the `worker` struct was cleaned up, removing several fields that were merely copied from the `ConnectorMessage`.

# v2.10.2 - Helptext Formatting Update
Now that robots can tell the difference between being addressed using it's name or it's alias, the help system has been updated:
* The old "(bot), do something" format can be replaced with "(alias) do something", so e.g. you'll get "Floyd, do something" in the first case, or ";do something" in the second case; this lets the robot's administrator provide help in a format the users are most likely to use
* Connectors now have a `FormatHelp` method that takes and returns a string, and allows the protocol to further modify the help line to be more visually appealing for the specific protocol; in Slack, for instance, the command portion of the help line is enclosed in single-backticks, so you get e.g. "`;build <package>` - start a new build of \<package\>"

# v2.10.1 - ChannelOnly matching
`CommandMatchers` and `MessageMatchers` can now set a boolean flag, `ChannelOnly`, to only match messages in a main channel or DM (not a thread).

# v2.10.0 - Command Mode, thread Subscribe API, short-term memory backing store
* Once again for the AI plugin, it's useful to differentiate between robot commands using the robot's single-character alias vs the robot's regular name. The single-character alias is mostly understood by users as a "command prefix" for the robot, and used with chatops commands for e.g. deploying software. Using the robot's name, however, could indicate the user intends the message to be delivered to the AI. Starting in this version, external plugins will see a new `GOPHER_CMDMODE` environment variable, which will be "alias", "name", or "direct", and can be used to modify behavior.
* This seems like an obvious API in retrospect - a plugin can now Subscribe to (and Unsubscribe from) the thread of the message that started it. This allows plugins to deal with conversations in a fairly natural fashion - starting a conversation thread and subscribing to it, and keeping state in short-term memory (which for threads is quite long).
* Thread-short-term memories and thread subscriptions both time out after one week; given that robots occasionally need to be redeployed, these memories are now backed by the robot's brain, to prevent amnesia after a rebuild. Short-term memories still expire, making them easier to use in most cases than the long-term memory API.

# v2.9.2 - Pipeline / Namespace Bugfix
It bears repeating here (not actually sure it's documented elsewhere) - both Jobs and Plugins can create pipelines, but simple tasks only inherit the namespace in a Job pipeline, where simple tasks are intended to have access to Job-specific memories and secrets. Plugin-namespaced memories and secrets should be restricted to the plugin only, largely because plugins also serve as authorizers (establishing group membership) and elevators (which normally require some form of 2fa).

# v2.9.1 - Slack Updates, Bugfixes and Security Enhancement
For the first time in years I'm running a robot that does NOT have `IgnoreUnlistedUsers: true` - and for the record, if you're using Gopherbot for team chat DevOps, I strongly recommend you make the effort. In any event, moving my personal robot Floyd to my local community Slack team has revealed some issues in the Slack connector with the regex that matches a username mention. In addition to widening that regex appropriately, I've updated the Slack library for a bugfix in the socketmode connector, and improved the security in user loading, adding a guardrail to prevent Slack-provided usernames from conflicting with users listed in the UserRoster.

# v2.9.0 - Ephemeral Memories Update
This is a large-ish change in the API and operation of ephemeral memories, once again motivated by the AI plugin and partly enabled by threaded conversation support:
* The old notion of "short term memories" is now more generically "ephemeral memories" - memories stored in RAM that are forgotten after a time
* Ephemeral memories for a channel are shorter-lived, about 7 minutes, reflecting that channel topics and contexts tend to change relatively quickly
* Ephemeral memories for a thread are longer-lived, almost two days, reflecting that threads tend to stay on a single topic and may last days
* Remember, RememberThread and Recall now take an optional third argument, shared; when a memory is shared, it is the same for all users in the channel/thread - but NOTE, there is no record locking for shared (or non-shared, for that matter) memories!
   * To keep multiple plugin runs from stomping on each other, you can use the Exclusive API
   * The Exclusive API method, formerly exclusive to Jobs and job pipelines, now works with Plugins, too

# v2.8.2 - Allow plugins to handle "robot?"
Very minor enhancement; if your robot's name was floyd, and the user typed "floyd?", the only way to handle it was to use a message matcher that hard-coded the robot's name. Now you can define a command with the regex '^\?$' to catch this case and handle it.  Added for - you guessed it - the AI plugin.

# v2.8.0, 2.8.1 - POTENTIALLY BREAKING ambient message matching update, direct catch-all update
In most cases, if someone directs a message to the robot (issues a "command") in a channel, the message should go to the CatchAll if no command matches. Up to now, the default behavior was to match against `MessageMatchers`, even in the case of a command. Starting with v2.8.0, commands will no longer match against message matchers unless `AmbientMatchCommand` is set `true`. For most configurations this shouldn't be an issue, but this could break some configurations. Note that direct messages to the robot are always commands, and will need `AmbientMatchCommand` set to match against a plugin's `MessageMatchers`; for the normal use case of `MessageMatchers` (e.g. mentioning Chuck Norris), this probably isn't what you want.

If you want a plugin to override the help system matching direct messages that don't match a command, you can create an alternate configuration for the plugin with `CatchAll: true` and `DirectOnly: true`, which will mark it as a "specific" catch-all.

# v2.7.4 - Longer Memories
Short-term memories really only came in to serious use with the OpenAI plugin, where 7 minutes just wasn't enough for expectations. Now Gopherbot robots have a 14-hour short-term memory.

# v2.7.3 - Improved ruby/python3 dependency support
To make it sane to use a `Gemfile`, we now set `GEM_HOME=${HOME}/.local`, which causes bundler to install ruby gems in the same location as python3. To install ruby and python dependencies with the included `install-libs` job, create `Gemfile` and/or `requirements.txt` to the root of your robot's repository, and add this `ScheduledJob` to your `robot.yaml`:
```yaml
- Name: install-libs
  Schedule: "@init"
```

# v2.7.2 - Enhanced Catch-All Plugin Matching
Since the earliest versions, **Gopherbot** has had a `CatchAll` setting for plugins, allowing special plugins to be run when the robot is being spoken to (via direct message, or when mentioned by name) but no command matched. The only default plugin so configured is the built-in `help` plugin, which gives the familiar "No command matched ..." responses.

Previously the behavior of the `CatchAll` setting for a plugin wasn't very useful; if you set it to `true` for a plugin, and didn't disable it for the built-in `help` plugin, multiple plugins would match and result in an error log message with neither being run. Starting with v2.7.2, if you set `CatchAll: true` and your plugin is either marked `DirectOnly`, or the current channel is in the list of channels for the plugin, it will match as a *specific* catch-all. So long as only one specific catch-all matches, your plugin will be called, and only fall-back to the built-in `help` plugin if no specific catch-all matches. In detail:
* If a single *specific* catch-all matches, it will be called.
* If more than 1 *specific* catch-all matches, an error will be logged and nothing will be called.
* If no *specific* catch-all matched and only a single fallback catch-all matched, it will be called.
* If multiple fallback catch-alls match, an error will be logged and nothing will be called.

The only other small change is that previously the catch-all was called with the space-reduced version of the message, and going forward it will be called with the full message, which may include e.g. newlines.

This new behavior allows for:
* Channel-specific help messages
* Channel-specific and/or DM default plugins; the impetus for this enhancement was to allow direct messages to a robot to be routed to the AI plugin if no other command matched

# v2.7.1 - More regex updates, new global ignore command
* In another potential but unlikely breaking change, command matching now sets the `s` flag by default, which IMHO gives the expected result for e.g. `(?i:echo (.*))` - for a robot command, `.*` would match everything to the end of the message.
* Additionally, the message space collapsing now includes all whitespace, including newlines. Thus, for somewhat standard commands with simple string arguments, the command will still match even if the user breaks up a long command with a newline.
* Since I've been deploying the OpenAI chat plugin in an #ai channel, where EVERY message matches the plugin, I've added a simple global `i(gnore) <anything>` command that allows you to speak to other people in a channel without triggering a command or error.

# v2.7.0 - Support for OpenAI Chat Plugin, ambient message matching fixes

With the massive ChatGPT craze, and a website that was frequently unavailable, I just *had* to write a AI plugin for Gopherbot. Unfortunately, (or fortunately?) using this plugin revealed a whole host of niggly and corner-case bugs that have mostly been fixed in v2.6.4.1-5. There was one final fix, however, that could possibly break existing plugins...

## Change to `MessageMatchers` matching
Ambient `MessageMatchers` were previously matching against the original message with the robot name or alias stripped away, so:
* if your robot's alias was, for instance `;`
* and the user typed the command `;some garbage`
* and your plugin had a `MessageMatcher` of e.g. `^([^;].*)`
* ... the matcher would be checked against `some garbage`, and match even when it shouldn't

This has now been fixed, but since `MessageMatchers` now match differently (and the version string was getting kind of nutty), I went ahead and bumped all the way to v2.7.0.

## Multi-line match and space fixes
In the early early days of Gopherbot, users would frequently copy and paste in commands, and many times there would be multiple spaces between words - `something  like   this`. This wouldn't match the regex `/something like this/`, and meant regexes would need to be littered with `/s+`, which was crazy ugly. In a stroke of cleverness, I sanitized input by replacing all instances of multiple spaces with a single space, thus fixing that issue but good. Until, that is, I started caring about preserving those spaces when sending messages to plugins (think sending formatted Python code blocks to ChatGPT). To preserve spaces AND preserve space-collapsed matching, the engine first tries matching against the full message text, and failing that the space-collapsed version. If either matches, that text is passed to the plugin.

Additionally, the **terminal** connector was updated with a `GenerateNewlines` boolean option. When this option is set, you can type a command like: `;Complete this sequence:\n5,4,3,2`, and a message with a newline will be sent to the engine. Note, however, that `(.*)` won't always match the whole thing as you might expect - you can add the `s` flag to your regexp definition (e.g. `(?is:<something>)`) or use `([\s\S]*)` to match *everything*, including the newlines (CHANGED, see v2.7.1 above).

## The OpenAI Chat Plugin
If you'd like to try the `ai.rb` plugin, which uses the OpenAI *completions* api, you'll have to check out [**Floyd**](https://github.com/parsley42/floyd-gopherbot) (my oldest robot).

# v2.6.4(.1) - Add RememberThread, RememberContextThread; Message IDs, env vars
* To support development of a new OpenAI Chat plugin, these functions were needed to allow a command in a channel to create a short-term memory for a new thread created by the robot.
* Added a `Threaded` method for all libraries (except BASH, where you can just set `GOPHER_THREADED_MESSAGE`); `bot.Threaded()` will return a bot object that always speaks and acts in the message thread.
* Added the `GOPHER_MESSAGE_ID` and `GOPHER_START_MESSAGE_ID` environment variables with opaque message ids; mainly only useful for generating Slack links to a message in a thread, wheeee
* Added `GOPHER_CHANNEL_ID`, `GOPHER_USER_ID`, `GOPHER_START_CHANNEL_ID` env vars (.1)

# v2.6.3 - IDE, Development and Packaging Updates
* The dev container now defines GOPHER_IDE=true, which activates IDE mode:
  * This will always cause `gopherbot` to change directory to `$HOME` on startup, to prevent a common UX issue where admins start the robot in, say, the `custom/` directory; this behavior can only be overridden by explicity unsetting the env var in the shell: `$ unset GOPHER_IDE`
  * When `GOPHER_CUSTOM_REPOSITORY` is set in the initial container environment, `gopherbot` will set `GOPHER_PROTOCOL=terminal` and `GOPHER_BRAIN=mem` - sensible defaults for the IDE / development environment; this can be more easily overridden by passing the `-o` flag to `gopherbot` if you want to run your robot in production mode in the development environment: `$ gopherbot -o`
* The dev and min containers now include the `ruby-dev` and `python-dev` packages, allowing gems/packages that need to be compiled to be installed
* The ruby `debug` gem and python `rpdb` packages are installed in the dev container, allowing developers to set breakpoints in their robot scripts and attach a command-line debugger to the running script:
  * For Ruby, you can set a breakpoint by inserting `require debug/open` in the code, then running `$ rdbg -A` to attach when you see it hit the breakpoint in the output log
  * Python should offer similar functionality using `import rpdb` and `rpdb.set_trace()` to set the breakpoint in your code, then using `telnet` to attach
  * Keep in mind that each time a Ruby or Python extension is called, a new interpreter is spun up; thus the normal workflow is just inserting a breakpoint in the code and e.g. telling your robot to "do that again" - e.g. `> bot, trigger my buggy code`
* I've introduced experimental support for `GOPHER_ENVIRONMENT`, which can be used to have a production and development set of encrypted credentials, to allow developers to work with the robot without having full access to production credentials. If you do nothing at all, your script extensions will simply see two new environment variables:
  * `GOPHER_ENVIRONMENT`, which defaults to "production"
  * `GOPHER_BRAIN`, which is set to the name of the brain being used

To try out working with a separate environment:
  * Start an IDE with no robot configured (a pristine `bot.env`)
  * Open a terminal window in `/home/bot` and generate a new encryption key: `$ export GOPHER_ENCRYPTION_KEY=$(openssl rand -base64 24); echo $GOPHER_ENCRYPTION_KEY`
  * Start up the default robot, and as soon as you get the terminal connector prompt press `<ctrl-d>`
  * Copy the contents of `custom/binary-encrypted-key` to your production robot as `custom/binary-encrypted-key.<env>`, e.g. `custom/binary-encrypted-key.dev`, and push the new file
  * Copy your robot's profile to a new version with e.g. `GOPHER_ENVIRONMENT=dev`, and `GOPHER_ENCRYPTION_KEY=<value from above>`
  * When you start your robot from the new profile, it will use the new `GOPHER_ENCRYPTION_KEY` to decrypt e.g. `binary-encrypted-key.dev`, which will then be used for decrypting any secrets found in your robot's configuration files
  * It's up to you to add any needed `if` and `.Include` logic in your configuration files (testing the `GOPHER_ENVIRONMENT` environment variable) to make sure your robot can start with the alternate environment

# v2.6.2 - Boogs
* Fixed an old bug where the robot might not answer to being @mentioned in Slack. To be sure @mentions work, your robot needs to list itself in the Slack user map, with the same name as in `robot.yaml`.
* Fixed a lingering thread bug in the bash library
* Updated robot.skel and autosetup/addadmin plugins to automatically configure the bot correctly for recognizing a @mention; to update your robot, use the robot's built-in `info` command to get the ID, then add a stanza to your the UserRoster of your `slack.yaml`:
```yaml
  - UserName: bishop
    UserID: U0123456789
    BotUser: true
```
* Batch of improvements to the `cbot.sh` script for local dockerized development

# v2.6.1 - Robots can now talk to themselves
* When Gopherbot was first created, it didn't make much sense for Gopherbot robots to process messages that originated from themselves. Now that messages carry a "ThreadID" ("GOPHER_THREAD_ID"), a clever roboticist can take advantage of the robot's ability to hear itself to e.g. associate state with a thread. To enable this, set `HearSelf: true` in your Slack ProcotolConfig.

# v2.6.0 - New IDE and Threaded Conversations Support
* The documentation is just starting to catch up with the new **Gopherbot IDE** - a giant (>2G) container with everything you need to do development either on your robot, or Gopherbot itself.
* Gopherbot robots can now hear an opaque "ThreadID" (env var: "GOPHER_THREAD_ID") and "ThreadedMessage" (env var: "GOPHER_THREADED_MESSAGE", "true" when the message was issued in a thread). "Say" and "Reply" will post to the thread if the robot was spoken to there. Additionally, the new "SayThread", "ReplyThread" and "PromptThreadForReply" methods will create a new thread from a message in the channel, allowing you to keep noisy responses out of the main channel; the "help" command does this now.

# v2.5.2 - Socket mode stability and command standardization
* After a problem with a robot becoming unresponsive, an [issue](https://github.com/slack-go/slack/issues/1093) with the Slack Go library was found that could delay reconnecting in the event of a timeout on the websocket. This release incorporates a [fix](https://github.com/slack-go/slack/pull/1094).
*  To make job-related commands easier to remember, `<verb>-<noun>` synonyms were added for job related commands, e.g. `list-jobs`, `pause-job`, `resume-job`, etc.

# v2.5.1 - Bugfix Slack Plugin
This release just fixes a crash in the "slack id @user" plugin. It adds a type switch to differentiate the message objects returned by Socket Mode vs. RTM.

# v2.5.0 - Add socket mode support
This has been hanging over my head for a long while, so very happy to announce that starting with Gopherbot v2.5.0, robots can be configured with an app/bot token pair to connect via socket mode. Note that eventually Slack is expected to remove support for the old RTM protocol; as it is, creating new Gopherbot robots means ignoring some scary messages about using old/legacy/"classic" app support.

## Upgrading Existing Robots
1. You'll need to obtain a new set of credentials; the steps are fully documented in the [online manual](http://localhost:8888/botsetup/slacksock.html).
2. Use `gopherbot encrypt "..."` to create encrypted versions of these tokens (optionally, only the portion after the `xapp-` / `xoxb-`, although AES is supposed to be very resistant to partially-known plaintext attacks).
3. Edit your `conf/slack.yaml` and add lines for `AppToken` and `BotToken` under the `SlackToken` line; use the `decrypt` function to decrypt the encrypted values you created.

Now you can start your robot with it's new credentials. If all goes well, you can remove the `SlackToken`.

> NOTE: As far as Slack is concerned, this is a new app/user - so if you have a DM open with the old app, you should go ahead and close it.

> NOTE: The new version will still accept the old `SlackToken` to make legacy RTM connections.

# v2.4.9 - Run init jobs before other external scripts
This update allows plugins to take advantage of ruby gems and/or python modules installed from an init job. Prior to this update, the robot would need to be manually restarted, or plugins would have to be specially edited to require/import later in the script (ugh).

# v2.4.8 - Fix Slack escaping
The Slack "Raw" format was incorrectly escaping angle brackets ("<>"), disabling e.g. nice hyperlinks with text. This release corrects this issue, and also restores escaping of '*', '_', ':', '@', '#' and '`' for the "Variable" format - allowing these characters to "appear" unadulterated in message. I quote "appear" because there are invisible characters inserted - so copy/paste will get these. If the text is truly meant to be copied and pasted, use the "Fixed" format instead.

# v2.4.7 - Improve developer workflow
This release adds a new admin `change-branch <branch>` command to support more normal development workflows. Significant updates can be made to a robot on a development branch and tested / fixed before merging. In the event something breaks badly, `change-branch main` can quickly restore the robot to a working condition.

# v2.4.6 - Improve custom lib support
Bugfix release - fixes an issue with initial loading of plugins that require/import a library from `custom/lib`; this path wasn't included during plugin initial config loading, causing errors that resulted in plugins being disabled.

# v2.4.5 - Let the robot see angle brackets
This version always translates e.g. `&lt;` to `<`, since slack sends the former instead of the latter when the user in fact typed the latter. This allows the robot to 'see' angle brackets now, but unfortunately prevents the robot from seeing the string `&lt;`. Brackets seem more useful. (Same goes for `>`, naturally)

# v2.4.4 - File history fix, Channel Logging
(Possibly) **BREAKING CHANGE**: The `LoadableModules` configuration stanza is removed in this version. All releases since 2.1.0 have been fully static builds, with loadable modules disabled. Starting with this version, having `LoadableModules` defined will be a fatal error.

This update contains a fairly major fix - at some point the default file history provider was disabled, causing robots to fall back to memory-based history, which is of limited use. I didn't bother to bisect since the fix was pretty obvious and the culprit was definitely me (David).

The other big-ish change is the addition of channel logging. Using the `log channel (name)` command, administrators can log the raw strings from a channel to a file. This should help in a variety of situations where the bot administrator needs to see "what the robot sees" in a channel - for matching users, writing regular expressions, etc. Use `help debug` for command syntax.

# v2.4.3 - DynamoDB Brain support, more HOME updates
This update primary deals with correct operation for robots with non-file brains. It's noteworthy that DynamoDB is the best brain.
* During bootstrapping, the robot was always trying to restore a file-backed (git) brain; now the restore job handles non-file brains gracefully
* During configuration loading, `$HOME` was not being set, causing certain plugins to always be disabled when gems weren't found

# v2.4.2 - Improved IDE UX
This release makes a number of changes to the user experience for the Gopherbot IDE. Notably, it now copies the user's `$HOME/.gitconfig` to the container, so all user git settings apply.

# v2.4.1 - @init jobs
> **NOTE: Potentially breaking change**

This release adds a special '@init' value for scheduled jobs, allowing certain jobs to be run during start-up and reload. The use case for this was to allow decrypting an encrypted `.kube/config` file, but this could also be used for installing python packages and ruby gems.

The **potentially breaking change** is that `$HOME` is now fixed to `$GOPHER_HOME`, the main working directory for the robot. It doesn't make a lot of sense for this to be anything different, or to require this to be set elsewhere (e.g. in the `Dockerfile`).

# v2.4.0 - Gopherbot IDE
This is the first release of a mostly-working Gopherbot IDE, based on [Theia](https://theia-ide.org/). In short:
* Use the `gb-dev-profile` script to generate an IDE profile from a robot's `.env` file - this packages an ssh key and other **git** goodness into a new env file
* Use `gb-start-dev <profile>` to start a local docker container with the profile
* Once the robot starts, tell it to `start-ide` (or `start-theia`), then connect to `http://localhost:3000`

# v2.3.0 - ParameterSets
This release adds `ParameterSets`. Since the `NameSpace` of a task also determines access to long-term memories, tasks can only have a single namespace. ParameterSets are identical in form to NameSpaces, but a given job/task/plugin can list multiple ParameterSets. This allows more flexible sharing of secrets between jobs - especially for secrets like GITHUB_TOKENs, that don't really map well to a namespace. For examples of `ParameterSets` in action, see [Clu](https://github.com/parsley42/clu-gopherbot).

# v2.2.0
* Updated `go.mod` and import paths for (hopefully) 'proper' go module support; since **Gopherbot** is far more an application than a library, this should only matter for indexes like [pkg.go.dev](https://pkg.go.dev)

# v2.1.5 - TOTP Update, Improved help
* Stronger TOTP implementation; generate user codes with the CLI and add encrypted secrets to `conf/plugins/builtin-totp.yaml`
* Make standard "robot, help" contextual (less noisy); "robot, help-all" gives the formerly verbose output
* Export `PYTHONPATH` and `RUBYLIB`, removed ugly env-var references from python & ruby plugins
* Simple tasks now inherit their memory namespace from the pipeline they're added to

# v2.1.4 - Ephemeral Memories
* Allow for ephemeral memories with the git brain; any memories with leading underscores will be ignored by the '.gitignore', allowing developers to prevent frequent git commits for fast changing memories. Also made the .gitignore algorithm more robust.

# v2.1.3
* More container build updates - make all containers more useful for dev env

# v2.1.2
* Update container builds, using GitHub actions for releases and containers

# v2.1.1
* Compile in the dynamodb brain, famously used by Floyd
* Update aws and other deps

# v2.1.0 - Remove Go Modules
Changes since v2.0.2:

The major update for 2.1.0+ is the temporary and possibly permanent removal of modular builds. Support for loadable Go modules was *cool*, but it meant that I couldn't build a single distribution archive installable on all recent Linux distributions due to modular builds using the gcc toolchain linker. Starting with 2.1.0, **Gopherbot** will once again create installable build artifacts.

The other major change is a greater focus on **Slack** as the primary protocol. While the **terminal** connector will continue to be the best way to develop Gopherbot jobs and plugins, the primary build artifacts will be largely Slack-specific.

Other updates/fixes:
* Several fixes to the `autosetup` installer for additional robustness.
