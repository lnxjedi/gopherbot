# Gopherbot Development Notes

`DevNotes.md` - TODO items and design notes for future development.

## Encrypted Brain
The new EncryptedBrain type will take and return a struct instead:
 Magic: int = 0xDeadBeef - for validating the data structure
 Memory: []byte - encrypted contents of the memory

* File & Dynamo brains will get new functions and register a provider for an EncryptedBrain
* When Convert is True, the retrieve function will read, validate, then encrypt & store if memory was unencrypted before returning
* Add admin command 'convert <key>' to forces the robot to read the memory and re-store
* Robot should take a new EncryptedBrain parameter, exclusive with Brain
* BrainKey is optional, can be specified at runtime
* The BrainKey should unlock the 'real' key, for later re-keying if desired, e.g. if the admin switches from a configured key to a runtime-supplied key

### Datum Processing
Datum are protected by serializing all access to memories through a select loop. It should be reviewed / rewritten:
* Replace "seen" with a timestamp
* Guarantee 2 seconds exclusive access
* Lower the brain cycle to 0.1s, guaranteeing 2.0-2.1s access to a memory

## Plugins and Jobs

### TODO
* Support for Go jobs

### Augmentations
Plugins and jobs share a common botCaller struct. Both can have a defined NameSpace,
which determines sharing of long-term memories and environment variables (parameters).

Plugins can now also reference parameters which are set as environment variables. An example might be credentials that an administrator sets with the 'set environment ...' built-in.

#### NameSpaces
Long-term memories have always been scoped by plugin name, so when "rubydemo" stores a long-term memory called e.g. `preferences`, internally the key has been set to `rubydemo:preferences`. Now, both plugins and jobs will be able to explicitly set a `NameSpace`, which determines what other jobs/plugins share the long-term memories. Also, environment variables will be scoped by namespace, and organized under a `bot` top-level namespace, e.g. `bot:environment:rubydemo:secretkey`.

The `bot` top-level namespace will be an illegal name for a job or plugin.

Jobs or plugins that don't explicitly set `NameSpace` will default to NameSpace==job or plugin name.

## Pipelines

Jobs and Plugins can queue up additional jobs/tasks with AddTask(...), and if the job/plugin exits 0, the next task in the list will be run, subject to security checks.

## Histories

Whenever a new pipeline starts, if the initiating job/plugin has HistoryLogs > 0, a history file will be recorded, tagged with the name of the job/plugin.

### How Things Work

* Everything the robot does will be a pipeline; most times it will just be a single job or plugin run
* Every item in the pipeline will be a job or plugin defined in `gopherbot.yaml`
* Builtins will be augmented with commands for listing all current pipelines running, which can possibly be canceled (killed). Certain builtins will be allowed to run even when the robot is shutting down:
  * Builtins that run almost instantly
  * Builtins that read internal data but don't start new pipelines
  * Builtins that report on running pipelines or allow aborting or killing/cancelling pipelines
* The `Robot` object, created at the start of a pipeline, will take on a more important role of carrying state through the pipeline; in addition to other struct data items, it will get a `runID` incrementing integer for each job/plugin
* When a pipeline starts a pointer to the Robot will be stored in a global table of running pipelines
* Pipelines can be started by:
  * A message or command matching a plugin
  * A scheduled job running
  * A job being triggered manually with the `run job...` builtin (subject to security checks)
* The Robot object will carry a pointer to the current plugins and jobs
* In addition to the pluginID hash, there will be runID, a monotonically incrementing int that starts with a random int and is OK with rollovers
  * runIDs will be initilized and stored in RAM only
  * The Robot will carry a copy of the runID
  * The runID will be passed to plugins and returned in method calls to identify the Robot struct
  * A global hash indexed by pluginID and runID will be set before the first job/plugin in a pipeline is run
  * The global hash will be used to get the Robot object for the pipeline in http method calls
  * The Robot needs a mutex to protect it, since admin commands can read from the Robot while a pipeline is running
* The Robot will also get a historyIndex:
  * The historyIndex is only needed when the 
* If the job/plugin configures a Next*, it will be set in the Robot
* SetNextJob|Plugin(...) can change the Next*
  * AddTask(...) will replace CallPlugin

### Security

#### Change in the model

Previously, Gopherbot design tried to provide some separation between plugins, to protect local plugins from third-party. This added extra complexity to configuration, which leads to brittleness. For Gopherbot to become more of a DevOps bot capable of running scheduled jobs, and with built-in knowledge of pipelines, security between plugins has been mostly removed. NameSpace support provides some level of separation, but mainly so that sharing between jobs/plugins is intentional and not accidental.

Security now focuses mainly on determining which users can run which plugins and start which jobs, and in what channels (for visibility of actions).

Documentation on elevation and authorization should be clarified:

#### Authorization

`Authorization` answers the question "is this user allowed to perform this action?" This is normally done by checking group membership of some sort (using the value of `AuthRequire`), but could also be based on values for command and its arguments.

#### Elevation

`Elevation` answers the question "how certain is it that this request REALLY came from the user, and not somebody else who found an unlocked chat session?" This is best accomplished by an easy two-factor method such as providing an OTP code or using Duo.

#### How Things Work

##### Scheduled Jobs / Plugins

Scheduled jobs configured in `gopherbot.yaml` should never trigger elevation or authorization checks. A `bypassSecurity` bool in the created Robot object should indicate this.

##### Interactive Jobs

Jobs run interatively with the `run job ...` built-in will Elevate if an Elevator is configured, and Authorize if an Authorizer and AuthRequire are set.

For jobs triggered interactively, authorization and elevation will be evaluated for each job in the pipeline.
* Elevation will only ever be checked once in a pipeline, after which the `elevated` flag will be set in the Robot object for the pipeline; if elevation fails at any point the pipeline will stop with an error
* Authorization will be checked for the triggering user for each job/plugin in the pipeline; failure at any point will fail the pipeline with an error

#### Elevation
If a job/plugin requires elevation, the configured elevator for that job/plugin will run, and if Elevation succeeds 

### Differences

#### Parameters vs. Arguments

Plugins, as always, will take arguments starting with a command. Jobs will instead take
parameters, which get set as environment variables.
* Administrators can use a built-in 'set environment <scope> var=value' command, direct
message only, to set environment vars that will be attached to the Robot when the
pipeline starts. The `global` scope will be set for all jobs & plugins, otherwise scope is a NameSpace.
* Plugins and Jobs can use a SetSessionParameter method that sets an environment variable in the current plugin, and in the Robot object for future jobs/plugins in the pipeline

#### Requirements

Jobs can be normal scripts in ANY scripting language, and need not import or
use the bot library. Jobs don't get called with "configure" and "init" the way
a plugin does.

#### Triggers

Plugins are triggered when a user issues a command or sends a message matching
a plugins CommandMatchers or MessageMatchers. Jobs can be triggered in one of
three ways:
* Configuring a `ScheduledJob` in `gopherbot.yaml`
* A user using the `run job ...` builtin command
* Another bot or integration triggers the job by matching one of the job's
  `Triggers`

#### Configuration

## 2.0 Goals
* Job scheduling
* Pipelines
* Remote plugin execution over ssh

## Questions

Can a plugin schedule a job? E.g. for the time tracking plugin; it could be written to run in the background, or it could schedule a job that checks back with the user periodically.

Are all commands checked before all message matchers? Multiple matching commands should do nothing, but what about multiple message matchers? Probably, all matching message matchers should run. If a command matches, message matchers probably should not be checked, as that might cause "side effects".

## TODO Items

- Plugin debugging verbose - emit messages for user message matching(?); when the user requesting debugging sets verbose, message match checks should trigger debug messages as well if the plugin being debugged has message matchers
- Slack connector: look up new Bot IDs on the fly if a new bot ID is seen; need to add locking of the bots[] map
- Move slack send loop into anon func in Run
- Consider: Use globalLock for protocolConfig, brainConfig, elevateConfig
- Add SendBotMessage() and GetBotMessage() methods for Slack in a conditional
  compilation connector_testing.go ala GetEvents - see emit_testing.go; use for
  connecting to slack and immediately quitting
- Documentation for events, emit, development and testing - Developing.md
- Stop hard-coding 'it' for contexts; 'server:it:the server' instead of
  'server', (context followed by generics that match); e.g. "reboot server",
  "reboot the server", "reboot it" would all check the "server" short-term
  memory to get e.g. "webtest1.example.com"
- Add more sample transcripts to documentation using the terminal connector
- The 'catchall' plugin should be in gopherbot.yaml (There Can Be Only One)
- Consider: should configuration loading record all explicit values, with
  intentional defaults when nothing configured explicitly? (probably yes,
  instead of always using the zero-values of the type)
- Re-sign psdemo and powershell lib (needs to be done on Windows)