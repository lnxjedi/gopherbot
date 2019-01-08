# Gopherbot Development Notes - Current

`DevNotes.md` - TODO items and design notes for future developmen

## 2.0 Goals
* Job scheduling (DONE)
* Pipelines (DONE)
* Remote plugin execution over ssh (ON HOLD)
* GopherCI for CI/CD (IN PROGRESS)

## Version 2 Stable Release TODOs
These are the items deemed as required for releasing version 2 because they change fundamental operation, configuration, or APIs, or to address stuff that's broken.
- DONE: Update the Ansible playbook for protected install / suid gopherbot
- DONE: Docker images w/ suid robot gopherbot running in protected zone; sample makefile / scripts for creating docker images named after the robot; move docker key to separate task
- DONE: Add 'TriggersOnly' flag for robot users that can only match triggers; ref: Great Chuck Norris War of 2018

## Version 2 Final Release TODOs
- TODO: Security:
  - DONE?: Squelch arguments in debugging messages when it's a DM - audit
  - Return to drop priv full when thread OS state propagation fixed; issue #29613
- TODO: Bugs & buglets:
  - Squelch Replies to 'TriggersOnly' users - use Say "UserName: foo"; slack webhooks can't be @mentioned. Easiest solution - add TriggersOnly flag to Robot that's set in makeRobot and cloned. 
- TODO: Add tests for:
  - Pipelines
  - Triggers only
  - Pipeline failures
  - Users listed in the UserRoster w/ alt names
  - Test repository configured parameters
  - Decrypting values encrypted in yaml files
- TODO: Write documentation for:
  - Need for BotInfo to provide the robot with a Name and Email
  - Pipelines
  - Namespaces / extended namespaces and histories; histories include branch, namespaces don't
  - store repository parameter can include a branch - TEST; e.g. store repository parameter xxx/master would override value for xxx in branch master
  - CI/CI
  - repositories.xml
  - UserRoster
  - TriggersOnly & cross-robot triggering

### Wishlist
These items aren't required for release, but desired soonish
- TODO: (maybe later) clean up IncomingMessage / botContext struct to eliminate dupes from the ConnectorMessage
- TODO: (f) skip to final (failed) task for history; may need to modify Section history breaks for non-primary pipeline tasks
- TODO: Add tests that check behavior of UserRoster / attributes, user w/ no username, etc.
- TODO: Add admin "monitor \<channel\>" / "stop monitoring" to DM admin with all messages to a channel similar to debug, for use in plugin devel & troubleshooting
- TODO: Update 'Starting job xxxx' message to include arguments; e.g. 'Starting job localtrusted github.com/lnxjedi/gopherbot master'
- TODO: Ansible playbook / Dockerfiles: /var/lib/gopherbot/(.env, custom) gopherbot:550:bin:root; custom:700:robot:robot; /opt/gopherbot/gopherbot suid robot
- TODO: CommandOnlyUsers - to allow bots to talk to each other without matching ambient messages; 
- TODO: Decrypt brain utility for removing 2nd level of encryption (secrets still encrypted)

## Protecting Secrets

This section outlines potential means of providing secrets to the robot in a secure manner, such that even malicious external scripts / plugins / jobs would not be able to obtain the Slack token or encryption key.

### SUID robot executable

This is the most promising.

Create e.g.:
* $GOPHER_INSTALLDIR/custom, root:root:500
* custom/repo, robot:robot:700 - location of robot configuration repository
* custom/brain, robot:robot:700 - location for robot file brain
* custom/.env, robot:robot:400 - starting environment

* Docker WORKDIR=$GOPHER_INSTALLDIR/custom
* systemd WorkingDirectory=$GOPHER_INSTALLDIR/custom
* /opt/gopherbot/gopherbot runs SUID robot

TODO:
* Update calling convention in runtasks.go to always use "/dev/stdin"
* DONE - GopherCI needs GetRepoData() to run in WorkSpace
* DONE - Return interpreter args - and use them - in getInterpreter; e.g. `#!/bin/bash -e` should return `/bin/bash`, [ "-e" ]
* DONE - Add "Protected" flag for jobs that run in configpath instead of workspace
* DONE - When relpath == true, run tasks by connecting script to stdin and running `<interpreter> /dev/stdin args`, otherwise use usual method
* DONE - Mark `update` job as protected and test
* DONE - Load gopherbot.env from configpath, in both start_* and conf.go
* DONE - Integrate godotenv for loading environment from $cwd/.env & $cwd/gopherbot.env
* DONE - Update startup to allow for relative path to repo & brain
* DONE - Put workings in to allow config repo update to happen in the jail
* DONE - Prevent scripts/plugins that are NOT update from setting the working dir relative to cwd

Add to documentation:
#### Running a robot with the 'term' connector
In many cases you can develop plugins and test your robot with the simple `terminal` connector. For developing tests, use `make test` to build a binary with Event gathering and display enabled. To run the robot, `cd` to the configuration directory and e.g.:
```shell
$ GOPHER_PROTOCOL=term path/to/gopherbot -l /tmp/term.log
```
The test suite configuration directories set the protocol and log file in `gopherbot.env`, so with a test configuration directory you can simply use `/path/to/gopherbot`.

### Environment Scrubbing
Gopherbot should make every attempt to prevent credential stealing by any potential rogue code. One ugly 'leak' of sensitive data comes from `/proc/\<pid\>/environ` - once a process starts, the contents of `environ` remain unchanged, even after e.g. `os.Unsetenv(...)`. One way to scrub the environment is to use `syscall.Exec`:
```go
package main

import (
	"bufio"
	"fmt"
	"os"
	"syscall"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	if len(os.Environ()) > 0 {
		fmt.Println("Scrubbing the environment, check my PID/environ and press <enter> to start")
		fmt.Print("-> ")
		_, _ = reader.ReadString('\n')
		self, _ := os.Executable()
		syscall.Exec(self, []string{}, []string{})
	}
	os.Setenv("SCRUBBED", "true")
	fmt.Println("Done. Check PID/environ and press <enter> to finish")
	fmt.Print("-> ")
	_, _ = reader.ReadString('\n')
}
```
When the program is run and prints `Done. ...`, `/proc/\<pid\>/environ` is empty. This algorithm is incomplete, though - now NONE of the environment is available to the new process. The 'fix' for this (needs to be written/tested) is for the initial process to spawn a goroutine that re-runs `gopherbot` with the single argument `export`, and keeps an open FD reading from stdout. The main process can then call `syscall.Exec(...)` and read the env from the FD in to potentially a memguard LockedBuffer, where it can be safely dealt with.

## Robot Configuration
The stuff here needs to be checked / verified. Some may already be done.
To simplify locking:
* Some of the content of gopherbot.yaml should only be processed when the robot first starts, e.g. the brain, connector, listening port, logger
* Start-up items should be stored globally and readable without a lock
* These start-up items should be processed during initbot
* Items that can change on reload should be stored in a config struct similar to plugins and jobs; when the botContext is registered, it should get a copy of this struct that doesn't change for the life of the context, just like the task list
* Items that are processed to binary representations (e.g. string to loglevel) should be stored in non-public struct members; e.g. LogLevel(string) and logLevel(int)

## TODOs:
- DONE/TEST/DOCUMENT: fix configuration merging to include plugin default config
- DONE/TEST/DOCUMENT: Make sure plugins/tasks/jobs can be disabled in gopherbot.yaml
- TODO: Move *.sample files to new resources/custom - sample remote config repository
- TODO: Evaluate / update Ansible role to simplify use
- TODO: Low Priority - Support for Go jobs / tasks (can wait 'til first need)
- DONE/DOCUMENT: try converting lists of tasks/jobs/plugins to map[name]struct
- To obtain the `BXXXXXXXX` ID for a webhook, visit the settings page, e.g.: https://myteam.slack.com/services/BC7DLEPA8 <- that's it!
- TODO: Low Priority - Audit reload cases where something not configured needs to be un-configured, similar to scheduled jobs n->0
- TODO: Add WorkSpace to output from `info`
- TODO: paging for show log similar to history; same 2k page size, but showing in reverse (e.g. 'p' for previous page, 'q' to quit)
- TODO: Low Priority - Brain optimization; keep a global seenCache map[string]bool:
	- if the entry exists, it's been seen and the bool indicates whether the memory exists
	- if the entry doesn't exist, check the memory and store whether it exists
	- when storing a memory, always store that the memory exists
	- best of all, the brain loop should mean no mutex!
- TODO/VERIFY: (This may be complete) Audit use of 'debug' - most calls should be debugT? (before bot.currentTask is set)
- TODO: (LP) Move slack send loop into anon func in Run
- TODO: Documentation for events, emit, development and testing - Developing.md
- DONE/DOCUMENT: Stop hard-coding 'it' for contexts; 'server:it:the server' instead of
  'server', (context followed by generics that match); e.g. "reboot server",
  "reboot the server", "reboot it" would all check the "server" short-term
  memory to get e.g. "webtest1.example.com"
- TODO: This needs more thought - the 'catchall' plugin should be in gopherbot.yaml (There Can Be Only One)
- TODO/VERIFY: Plugin debugging verbose - emit messages for user message matching(?); when the user requesting debugging sets verbose, message match checks should trigger debug messages as well if the plugin being debugged has message matchers
- TODO: Re-sign psdemo and powershell lib (needs to be done on Windows)
- DONE: Update ansible role to stop creating installed gopherbot.yaml, favoring version
  from install archive

## Consider
Stuff to think about:
- Consider: Use globalLock for protocolConfig, brainConfig, elevateConfig
- Consider: should configuration loading record all explicit values, with intentional defaults when nothing configured explicitly? (probably yes, instead of always using the zero-values of the type)
- Are all commands checked before all message matchers? Multiple matching commands should do nothing, but what about multiple message matchers? Probably, all matching message matchers should run.

# Historical/Completed
(This stuff can be removed later, but is left for now as a reminder of what needs to be documented)
- Fix history paging; goes to next page if no reply
- Update jobbuiltins / history to list namespaces
- Create `run` task for running a command/script in a repository
- When a task in a pipeline is a job:
  - If it's a different job, create a new botContext and call startPipeline with a parent argument != nil == pointer to current bot
  - Start with inheriting environment, user, channel, msg, and namespace; can re-visit what's inherited in child jobs later
- Let registerActive take a parent arg; if non-nil, registerActive will set parent and child members of bots while holding the lock
- Add WorkSpace top-level item to the robot; the r/w wdirectory where it clones repositories and does stuff; export GOPHER_WORKSPACE in environment
- Remove WorkDirectory from jobs; jobs should consult environment vars and call SetWorkingDirectory
- Ansible role should create WorkSpace and put it in the gopherbot.yaml
- Change Exclusive behavior: instead of just returning true when bot.exclusive, make sure tag matches, and return false if not and log an error; also fail if jobName not set
- Put .gopherci/pipeline.sh in repositories
- Create gitea-ci job triggered by messages in dev
  - Clone repository in e.g WorkSpace/ansible/deploy-gopherbot
  - SetParameter PIPE_STARTED=true
  - If .gopherci/pipeline.sh exists, run it with no args
  - pipeline.sh should call and check e.g. Exclusive deploy-gopherbot
  - pipeline.sh can add self with stages: AddTask $GOPHER_JOBNAME tests
  - Since PIPE_STARTED is true, it'll just run pipeline.sh with the arg
- Export GOPHER_JOBNAME
- Repositories like role-foo should just have pipelines that call several SpawnPipeline "$GOPHER_JOBNAME" ansible deploy-gopherbot
- Stop 'massaging' regexes; instead, collapse multiple spaces in to a single space before checking the message in dispatch

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
* Move HistoryLogs and WorkingDirectory to BotJob
  * Make "update" a job triggered by the update plugin
  * Initialize history on the first job in the initial pipeline that has HistoryLogs > 0
  * Create new botContext for sub-pipelines; if parent context has no history, the child context may start one
* Implement "SpawnPipeline" - create a new goroutine / botContext and run startPipeline; allow trigger / plugin to start multiple jobs in parallel whose success or failure doesn't affect the running pipeline
* Add SetWorkingDirectory(...) bool method, allow relative paths relative to pipeline WorkingDirectory
* Set loglevel to Debug and clean up redundant log entries in runtasks
* Audit / update logging & history sections for starting pipelines & sub-pipelines
* Add builtins for listing and terminating running tasks

Algorithm mock-up for what to do after callTask in runPipeline:
```go
package main

import (
	"fmt"
)

func main() {
	requeued := false
	for i := 0; i < 5; i++ {
		fmt.Printf("Index is %d\n", i)
		if i == 0 && !requeued {
			i--
			requeued = true
		}
	}
}
```

## Histories

Whenever a new pipeline starts, if the initiating job/plugin has HistoryLogs > 0, a history file will be recorded, tagged with the name of the job/plugin.

### TODO
* Add a *botConf member to the Robot
* Use confLock.RLock() in registerActive() to obtain a copy of config
* Modify bot methods to query config items from the config (e.g. in logging) without locks

## Encrypted Brain
TODO: Add this info to documentation
When the robot starts with an encrypted brain, it'll look for a provided key to
decrypt the "bot:brainKey" memory to retrieve the actual key used to en-/de-crypt memories.
* If brainKey doesn't exist, it will be randomly generated and encrytped with the provided key
* If it exists but can't be decrypted, encryptBrain will be true but cryptKey.initialized will be false
* Low-level brain functions can check the encryptBrain bool w/o locking
* cryptKey is protected by an RWLock so an admin can later provide a key for unlocking the brainKey
* When encryptBrain is true and cryptKey.initialized is true, a failure to decrypt a memory should be interpreted as an unencrypted memory that gets encrypted and stored, then returned
* Add admin command 'convert \<key\>' to forces the robot to read the memory and re-store
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
* Write 'rekey brain \<foo\>' admin commands

## Plugins and Jobs

### Augmentations
Plugins and jobs share a common botCaller struct. Both can have a defined NameSpace,
which determines sharing of long-term memories and environment variables (parameters).

Plugins can now also reference parameters which are set as environment variables. An example might be credentials that an administrator sets with the 'set environment ...' built-in.

#### NameSpaces
Changes to NameSpaces:
- Get rid of PrivateNameSpace - no more inheritance
- Add NameSpace to tasks so they can share with jobs, plugins
- Exclusive is for jobs only (1 per pipeline), so c.jobName must be set to use it

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
* Administrators can use a built-in 'set environment \<scope\> var=value' command, direct
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
