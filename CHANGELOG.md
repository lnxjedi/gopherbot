# v2.6.2 - Boogs
* Fixed an old bug where the robot might not answer to being @mentioned in Slack. To be sure @mentions work, your robot needs to list itself in the Slack user map, with the same name as in `robot.yaml`.
* Fixed a lingering thread bug in the bash library
* Updated robot.skel and autosetup/addadmin plugins to automatically configure the bot correctly for recognizing a @mention; to update your robot, use the robot's built-in `info` command to get the ID, then add a stanza to your the UserRoster of your `slack.yaml`:
```yaml
  - UserName: bishop
    UserID: U0123456789
    BotUser: true
```

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
