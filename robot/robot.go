package robot

import "bytes"

// AttrRet implements Stringer so it can be interpolated with fmt if
// the plugin author is ok with ignoring the RetVal.
type AttrRet struct {
	Attribute string
	RetVal
}

func (a AttrRet) String() string {
	return a.Attribute
}

/*
Message (in the engine) is passed to each task as it runs, initialized from the botContext.
Tasks can copy and modify the Robot without affecting the botContext.

Message is also the struct returned to Robot.GetMessage()
*/
type Message struct {
	User            string            // The user who sent the message; this can be modified for replying to an arbitrary user
	ProtocolUser    string            // the protocol internal ID of the user
	Channel         string            // The channel where the message was received, or "" for a direct message. This can be modified to send a message to an arbitrary channel.
	ProtocolChannel string            // the protocol internal channel ID
	Protocol        Protocol          // slack, terminal, test, others; used for interpreting rawmsg or sending messages with Format = 'Raw'
	Incoming        *ConnectorMessage // raw IncomingMessage object
	Format          MessageFormat     // The outgoing message format, one of Raw, Fixed, or Variable
}

// Robot defines the methods exposed by gopherbot.bot Robot struct, for
// use by plugins/jobs/tasks. See bot/Robot for complete definitions.
type Robot interface {
	// CheckAdmin returns true if the user is a configured administrator of the
	// robot, and true for automatic tasks. Should be used sparingly, when a single
	// plugin has multiple commands, some which require admin. Otherwise the plugin
	// should just configure RequireAdmin: true
	CheckAdmin() bool
	// Elevate lets a plugin request elevation on the fly. When immediate = true,
	// the elevator should always prompt for 2fa; otherwise a configured timeout
	// should apply. Elevate is similar to "sudo" in functionality, requiring the
	// user to provide additional proof of identity to perform an action.
	Elevate(bool) bool
	// GetBotAttribute returns an attribute of the robot or "" if unknown.
	// Current attributes:
	// name, alias, fullName, contact
	GetBotAttribute(a string) *AttrRet
	// GetUserAttribute returns a AttrRet with
	// - The string Attribute of a user, or "" if unknown/error
	// - A RetVal which is one of Ok, UserNotFound, AttributeNotFound
	// Current attributes:
	// name(handle), fullName, email, firstName, lastName, phone, internalID
	// TODO: supplement data with robot.yaml user's table, if an
	// admin wants to supplment whats available from the protocol.
	GetUserAttribute(u, a string) *AttrRet
	// GetSenderAttribute returns a AttrRet with
	// - The string Attribute of the sender, or "" if unknown/error
	// - A RetVal which is one of Ok, UserNotFound, AttributeNotFound
	// Current attributes:
	// name(handle), fullName, email, firstName, lastName, phone, internalID
	// TODO: (see above)
	GetSenderAttribute(a string) *AttrRet
	// GetTaskConfig unmarshals the job/plugin configuration into a struct.
	GetTaskConfig(cfgptr interface{}) RetVal
	// GetMessage returns a pointer to the robot.Message struct
	GetMessage() *Message
	// GetParameter retrieves the value of a parameter for a pipeline. Only useful
	// for Go plugins; external scripts have all the parameters for the pipeline exposed
	// as environment variables.
	GetParameter(name string) string
	Email(subject string, messageBody *bytes.Buffer, html ...bool) (ret RetVal)
	EmailUser(user, subject string, messageBody *bytes.Buffer, html ...bool) (ret RetVal)
	EmailAddress(address, subject string, messageBody *bytes.Buffer, html ...bool) (ret RetVal)
	/*
		Exclusive lets a plugin or job pipeline request exclusive execution, to prevent
		processes from stepping on each other when it's not safe, for instance when
		performing operations on a git repository in the filesystem.
		The string argument ("" allowed) is appended to the pipeline namespace
		to allow for greater granularity; e.g. two builds of different packages
		could use the same pipeline and run safely together, but if it's the same
		package the pipeline might want to queue or just abort. The queueTask
		argument indicates whether the pipeline should queue up the task to be
		restarted, or abort at the end of the current task. When Exclusive returns
		false, the current task should exit. Note that a plugin should only
		set queue to true under rare circumstances; most often it should use
		false and emit an error on failure.
		If the task requests queueing, it will either re-run (if the lock holder
		has finished) or queue up (if not) when the current task exits. When it's
		ready to run, the task will be started from the beginning with the same
		arguments, holding the Exclusive lock, and the call to Exclusive will
		always succeed.
		The safest way to use Exclusive is near the beginning of a pipeline.
	*/
	Exclusive(tag string, queueTask bool) (success bool)
	// Fixed is a convenience function for sending a message with fixed width
	// font.
	Fixed() Robot
	// MessageFormat returns a robot object with the given format, most likely for a
	// plugin that will mostly use e.g. Variable format.
	MessageFormat(f MessageFormat) Robot
	// Direct is a convenience function for initiating a DM conversation with a
	// user. Created initially so a plugin could prompt for a password in a DM.
	Direct() Robot
	// Threaded returns a robot associated with the thread of the incoming message.
	Threaded() Robot
	// Log logs a message to the robot's log file (or stderr) if the level
	// is lower than or equal to the robot's current log level
	Log(l LogLevel, m string, v ...interface{}) bool
	// SendChannelMessage lets a Go task easily send a message to an arbitrary
	// channel. Use Robot.Fixed().SendChannelMessage(...) for fixed-width
	// font.
	//
	// ch is the channel name (without the # prefix).
	// msg is the message text to send.
	// v is an optional list of arguments for formatting the message (like fmt.Printf).
	SendChannelMessage(ch, msg string, v ...interface{}) RetVal
	// SendChannelThreadMessage can send a message to a specific thread in a
	// channel.
	//
	// ch is the channel name (without the # prefix).
	// thr is the thread ID.
	// msg is the message text to send.
	// v is an optional list of arguments for formatting the message (like fmt.Printf).
	SendChannelThreadMessage(ch, thr, msg string, v ...interface{}) RetVal
	// SendUserChannelMessage sends a message to a user in a specific channel.
	// u - either an "<internalID>" or plain "username"
	// ch - either an "<internalID>" or plain "channelname"
	// msg - Go string with optional formatting
	// v ... - optional extra arguments for the format string
	SendUserChannelMessage(u, ch, msg string, v ...interface{}) RetVal
	// SendProtocolUserChannelMessage sends a message to a specific protocol and target.
	// protocol - connector name like "ssh" or "slack"
	// u - either an "<internalID>" or plain "username", or empty for channel send
	// ch - either an "<internalID>" or plain "channelname", or empty for DM
	// msg - Go string with optional formatting
	// v ... - optional extra arguments for the format string
	//
	// Semantics:
	// - non-empty u + empty ch => DM to user
	// - empty u + non-empty ch => message to channel
	// - non-empty u + non-empty ch => directed user-in-channel message
	// - empty u + empty ch => MissingArguments
	SendProtocolUserChannelMessage(protocol, u, ch, msg string, v ...interface{}) RetVal
	// SendUserMessage lets a plugin easily send a DM to a user. If a DM
	// fails, an error should be returned, since DMs may be used for sending
	// secret/sensitive information.
	// u - either an "<internalID>" or plain "username"
	// ch - either an "<internalID>" or plain "channelname"
	// thr - always the connector-provided threadID
	// msg - Go string with optional formatting
	// v ... - optional extra arguments for the format string
	SendUserChannelThreadMessage(u, ch, thr, msg string, v ...interface{}) RetVal
	// SendUserMessage lets a plugin easily send a DM to a user. If a DM
	// fails, an error should be returned, since DMs may be used for sending
	// secret/sensitive information.
	// u - either an "<internalID>" or plain "username"
	// msg - Go string with optional formatting
	// v ... - optional extra arguments for the format string
	SendUserMessage(u, msg string, v ...interface{}) RetVal
	// Reply directs a message to the user
	Reply(msg string, v ...interface{}) RetVal
	// ReplyThread directs a message to the user, creating a new thread
	ReplyThread(msg string, v ...interface{}) RetVal
	// Say just sends a message to the user or channel
	Say(msg string, v ...interface{}) RetVal
	// SayThread creates a new thread if replying to an existing message
	SayThread(msg string, v ...interface{}) RetVal
	// RandomInt uses the robot's seeded random to return a random int 0 <= retval < n
	RandomInt(n int) int
	// RandomString is a convenience function for returning a random string
	// from a slice of strings, so that replies can vary.
	RandomString(s []string) string
	// Pause is a convenience function to pause some fractional number of seconds.
	Pause(s float64)
	// PromptForReply lets a plugin direct a prompt string to a user and temporarily
	// register a regex for a reply expected to a multi-step command when the robot
	// needs more info. If the regular expression matches, it returns the matched
	// text and RetVal = Ok.
	// If there's an error getting the reply, it returns an empty string
	// with one of the following RetVals:
	//	 UserNotFound
	//	 ChannelNotFound
	//		Interrupted - the user canceled with '-'
	//	 UseDefaultValue - user supplied a single "=", meaning "use the default value"
	//		ReplyNotMatched - didn't successfully match for any reason
	//		MatcherNotFound - the regexId didn't correspond to a valid regex
	//		TimeoutExpired - the user didn't respond within the timeout window
	//
	// Plugin authors can define regex's for regexId's in the plugin's JSON config,
	// with the restriction that the regexId must start with a lowercase letter.
	// A pre-definied regex from the following list can also be used:
	//		Email
	//		Domain - an alpha-numeric domain name
	//		OTP - a 6-digit one-time password code
	//		IPAddr
	//		SimpleString - Characters commonly found in most english sentences, doesn't
	//	      include special characters like @, {, etc.
	//		YesNo
	//
	// In case it's not obvious, this is mainly only useful in a plugin where the robot
	// is prompting the user who issued the command.
	PromptForReply(regexID string, prompt string, v ...interface{}) (string, RetVal)
	// PromptThreadForReply is the same as PromptForReply, but it creates a new thread.
	PromptThreadForReply(regexID string, prompt string, v ...interface{}) (string, RetVal)
	// PromptUserForReply is identical to PromptForReply, but prompts a specific
	// user with a DM.
	PromptUserForReply(regexID string, user string, prompt string, v ...interface{}) (string, RetVal)
	// PromptUserChannelForReply is identical to PromptForReply, but prompts a
	// specific user in a given channel.
	PromptUserChannelForReply(regexID string, user, channel string, prompt string, v ...interface{}) (string, RetVal)
	// PromptUserChannelThreadForReply must be the single most unused API call in history, since
	// it would need to know the thread ID to begin with.
	PromptUserChannelThreadForReply(regexID string, user, channel, thread string, prompt string, v ...interface{}) (string, RetVal)
	// CheckoutDatum gets a datum from the robot's brain and unmarshals it into
	// a struct. If rw is set, the datum is checked out read-write and a non-empty
	// lock token is returned that expires after lockTimeout (250ms). The bool
	// return indicates whether the datum exists. Datum must be a pointer to a
	// var.
	CheckoutDatum(key string, datum interface{}, rw bool) (locktoken string, exists bool, ret RetVal)
	// CheckinDatum unlocks a datum without updating it, it always succeeds
	CheckinDatum(key, locktoken string)
	// UpdateDatum tries to update a piece of data in the robot's brain, providing
	// a struct to marshall and a (hopefully good) lock token. If err != nil, the
	// update failed.
	UpdateDatum(key, locktoken string, datum interface{}) (ret RetVal)
	// Remember adds a ephemeral memory (with no backing store) to the robot's
	// brain. This is used internally for resolving the meaning of "it", but can
	// be used by plugins to remember other contextual facts. Since memories are
	// indexed by user and channel, but not plugin, these facts can be referenced
	// between plugins.
	Remember(key, value string, shared bool)
	// RememberThread is identical to Remember, except that it forces the memory
	// to associate with the thread.
	RememberThread(key, value string, shared bool)
	// RememberContext is a convenience function that stores a context reference in
	// short term memories. e.g. RememberContext("server", "web1.my.dom") means that
	// next time the user uses "it" in the context of a "server", the robot will
	// substitute "web1.my.dom". Note: contexts have seen little love, so YMMV.
	RememberContext(context, value string)
	// RememberContextThread is identical to RememberContext, except that the memory
	// is forced to associate with the thread.
	RememberContextThread(context, value string)
	// Recall recalls a short term memory, or the empty string if it doesn't exist.
	// Note that there are no RecallThread methods - Recall is always in the current
	// context.
	Recall(key string, shared bool) string
	// SpawnJob creates a new pipeContext in a new goroutine to run a
	// job. It's primary use is for CI/CD applications where a single
	// triggered job may want to spawn several jobs when e.g. a dependency for
	// multiple projects is updated.
	SpawnJob(string, ...string) RetVal
	// AddTask puts another task (job or plugin) in the queue for the pipeline. Unlike other
	// CI/CD tools, gopherbot pipelines are code generated, not configured; it is,
	// however, trivial to write code that reads an arbitrary configuration file
	// and uses AddTask to generate a pipeline. When the task is a plugin, cmdargs
	// should be a command followed by arguments. For jobs, cmdargs are just
	// arguments passed to the job.
	AddTask(string, ...string) RetVal
	// FinalTask adds a task that always runs when the pipeline ends, whether
	// it succeeded or failed. This can be used to ensure that cleanup tasks like
	// terminating a VM or stopping the ssh-agent will run, regardless of whether
	// the pipeline failed.
	// Note that unlike other tasks, final tasks are run in reverse of the order
	// they're added to deal with common types of dependency, e.g. startA, startB,
	// stopB, stopA - where e.g. A doesn't stop cleanly unless B already stopped.
	FinalTask(string, ...string) RetVal
	// FailTask adds a task that runs only if the pipeline fails. This can be used
	// to e.g. notify a user / channel on failure.
	FailTask(string, ...string) RetVal
	// AddJob puts another job in the queue for the pipeline. The added job
	// will run in a new separate context, and when it completes the current
	// pipeline will resume if the job succeeded.
	AddJob(string, ...string) RetVal
	// AddCommand adds a plugin command to the pipeline. The command string
	// argument should match a CommandMatcher for the given plugin.
	AddCommand(string, string) RetVal
	// FinalCommand adds a plugin command that always runs when a pipeline
	// ends, for e.g. emailing the job history. The command string
	// argument should match a CommandMatcher for the given plugin.
	FinalCommand(string, string) RetVal
	// FailCommand adds a plugin command that runs whenever a pipeline fails,
	// for e.g. emailing the job history. The command string
	// argument should match a CommandMatcher for the given plugin.
	FailCommand(string, string) RetVal
	// RaisePriv lets go plugins raise privilege for a thread, allowing filesystem
	// access in GOPHER_HOME.
	RaisePriv(string)
	// SetParameter sets a parameter for the current pipeline, useful only for
	// passing parameters (as environment variables) to tasks later in the pipeline.
	SetParameter(string, string) bool
	/*
		SetWorkingDirectory sets the working directory of the pipeline for all external
		job/plugin/task scripts executed. The value of path is interpreted as follows:
		  - "/absolute/path" - tasks that follow will start with this workingDirectory;
		    "cleanup" won't work, see tasks/cleanup.sh (unsafe)
		  - "relative/path" - sets workingDirectory relative to baseDirectory;
		    workSpace or $(pwd) depending on value of Homed for the job/plugin starting
		    the pipeline
		  - "./sub/directory" - appends to the current workingDirectory
		  - "." - resets workingDirectory to baseDirectory

		Fails if the new working directory doesn't exist
		See also: tasks/setworkdir.sh for updating working directory in a pipeline
	*/
	SetWorkingDirectory(string) bool
}
