# Gopherbot Development Notes

`DevNotes.md` - TODO items and design notes for future development.

## Pipelines

Jobs and Plugins can queue up additional jobs/tasks with AddTask(...) and FinalTask(...), and if the job/plugin exits 0, the next task in the list will be run, subject to security checks. FinalTask() is for cleanup; all FinalTasks run regardless of the last exit status, and without security checks.

### Pipeline algorithm model

When a task in the middle of the pipeline adds tasks, it creates a new pipeline; when a task at the end of the pipeline adds tasks, they're just added to the end
of the currently running pipeline.

```go
package main

import (
	"fmt"
)

func runPipe(p []int) {
	fmt.Println("Running pipeline")
	l := len(p)
	n := make([]int, 0)
	for i := 0; i < l; i++ {
		switch i {
		case 2:
			n = append(n, 7, 8)
		case 4:
			n = append(n, 10, 11, 12)
		}
		t := len(n)
		fmt.Printf("Task: %d, index %d, added: %d\n", p[i], i, t)
		if t > 0 {
			if i == l - 1 {
				fmt.Println("Appending...")
				p = append(p, n...)
				l += t
			} else {
				fmt.Println("Running new pipeline...")
				runPipe(n)
			}
			n = []int{}
		}
	}
	fmt.Println("End pipeline")
}

func main() {
	t := []int{ 0, 1, 2, 3, 4 }
	runPipe(t)
	fmt.Println("Hello, playground")
}
```

### How Things Work

* Everything the robot does will be a pipeline; most times it will just be a single job or plugin run
* Every item in the pipeline will be a job, plugin, or task defined in `gopherbot.yaml`
* Builtins will be augmented with commands for listing all current pipelines running, which can possibly be canceled (killed). Certain builtins will be allowed to run even when the robot is shutting down:
  * Builtins that run almost instantly
  * Builtins that read internal data but don't start new pipelines
  * Builtins that report on running pipelines or allow aborting or killing/cancelling pipelines
* The `botContext` object, created at the start of a pipeline, will take on a more important role of carrying state through the pipeline; in addition to other struct data items, it will get a `runID` incrementing integer for each job/plugin
* When a pipeline starts a pointer to the Robot will be stored in a global table of running pipelines
* Pipelines can be started by:
  * A job `Trigger`
  * A message or command matching a plugin
  * A scheduled job running
  * A job being triggered manually with the `run job...` builtin (subject to security checks)
* New jobs started from Triggers, Scheduled tasks, or `run job ...` will take arguments
* The Robot object will carry a pointer to the current plugins and jobs
* In addition to the pluginID hash, there will be runID, a monotonically incrementing int that starts with a random int and is OK with rollovers
  * runIDs will be initilized and stored in RAM only
  * The Robot will carry a copy of the runID
  * The runID will be passed to plugins and returned in method calls to identify the Robot struct
  * A global hash indexed by pluginID and runID will be set before the first job/plugin in a pipeline is run
  * The global hash will be used to get the Robot object for the pipeline in http method calls
  * The Robot needs a mutex to protect it, since admin commands can read from the Robot while a pipeline is running
* The Robot will also get a historyIndex array for each job with histories:
  * The historyIndex is only needed when the number of histories kept is > 0
  * It has a monotically increasing int of every run of the pipeline
  * It has a datestamp
  * It has a record of the args to the pipeline
* AddTask(...) will replace CallPlugin

### TODO
* Add Exclusive(string) bool method so jobs can use `if Exclusive(...)` to exit gracefully if an exclusive pipeline is already running (for jobs that would step on themselves if run in parallel)
* Add SetWorkingDirectory(...) bool method, allow relative paths relative to GOPHER_CONFIGDIR
* Set loglevel to Debug and clean up redundant log entries in runtasks
* Audit / update logging & history sections for starting pipelines & sub-pipelines
* Add builtins for listing and terminating running tasks

## Histories

Whenever a new pipeline starts, if the initiating job/plugin has HistoryLogs > 0, a history file will be recorded, tagged with the name of the job/plugin.

## Robot Configuration
To simplify locking:
* Some of the content of gopherbot.yaml should only be processed when the robot first starts, e.g. the brain, connector, listening port, logger
* Start-up items should be stored globally and readable without a lock
* These start-up items should be processed during initbot
* Items that can change on reload should be stored in a config struct similar to plugins and jobs; when the botContext is registered, it should get a copy of this struct that doesn't change for the life of the context, just like the task list
* Items that are processed to binary representations (e.g. string to loglevel) should be stored in non-public struct members; e.g. LogLevel(string) and logLevel(int)

### TODO
* Add a *botConf member to the Robot
* Use confLock.RLock() in registerActive() to obtain a copy of config
* Modify bot methods to query config items from the config (e.g. in logging) without locks

## Encrypted Brain
When the robot starts with an encrypted brain, it'll look for a provided key to
decrypt the "bot:brainKey" memory to retrieve the actual key used to en-/de-crypt memories.
* If brainKey doesn't exist, it will be randomly generated and encrytped with the provided key
* If it exists but can't be decrypted, encryptBrain will be true but cryptBrain.initialized will be false
* Low-level brain functions can check the encryptBrain bool w/o locking
* cryptBrain is protected by an RWLock so an admin can later provide a key for unlocking the brainKey
* When encryptBrain is true and cryptBrain.initialized is true, a failure to decrypt a memory should be interpreted as an unencrypted memory that gets encrypted and stored, then returned
* Add admin command 'convert <key>' to forces the robot to read the memory and re-store
* Robot should take a new EncryptBrain bool parameter
* BrainKey is optional, can be specified at start or runtime
* The BrainKey should unlock the 'real' brainKey, for later re-keying if desired, e.g. if the admin switches from a configured key to a runtime-supplied key

### Datum Processing
Datum are protected by serializing all access to memories through a select loop. It should be reviewed / rewritten:
* Replace "seen" with a timestamp
* Guarantee 2 seconds exclusive access
* Lower the brain cycle to 0.1s, guaranteeing 2.0-2.1s access to a memory

### TODO
* Write reKey function
* Write 'rekey brain <foo>' admin commands

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

Scheduled jobs configured in `gopherbot.yaml` should never trigger elevation or authorization checks. An `automaticTask` bool in the created Robot object should indicate this.

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

Are all commands checked before all message matchers? Multiple matching commands should do nothing, but what about multiple message matchers? Probably, all matching message matchers should run.

## TODO Items

This is just a gathering spot for somewhat uncategorized TODO items...

- Audit use of 'debug' - most calls should be debugT? (before bot.currentTask is set)
- Plugin debugging verbose - emit messages for user message matching(?); when the user requesting debugging sets verbose, message match checks should trigger debug messages as well if the plugin being debugged has message matchers
- Plugin debugging channel option - ability to filter by channel if channel set
- Slack connector: look up new Bot IDs on the fly if a new bot ID is seen; need to add locking of the bots[] map
- Move slack send loop into anon func in Run
- Consider: Use globalLock for protocolConfig, brainConfig, elevateConfig
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
- Stop 'massaging' regexes; instead, collapse multiple spaces in to a single space before checking the message in dispatch