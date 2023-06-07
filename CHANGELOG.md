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
