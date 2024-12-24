// Package robot defines interfaces and constants for Go plugins, jobs and tasks
package robot

//go:generate stringer -type=TaskRetVal robot_constants.go

//go:generate stringer -type=RetVal robot_constants.go

//go:generate stringer -type=Protocol connector_defs.go

//go:generate stringer -type=MessageFormat robot_constants.go

//go:generate stringer -type=LogLevel robot_constants.go

// Generate String method with: go generate ./robot/

// LogLevel for determining when to output a log entry
type LogLevel int

// Definitions of log levels in order from most to least verbose
const (
	Trace LogLevel = iota
	Debug
	Info
	Audit // For plugins to emit auditable events
	Warn
	Error
	Fatal
)

/*
	TaskRetVal is an integer type for return values from Go tasks, plugins and

jobs. Handlers should return one of these.
*/
type TaskRetVal int

const (
	// Normal exit is for non-auth/non-elevating plugins and pipeline tasks; since this is the
	// default exit value, we don't use it to indicate successful authentication
	// or elevation. Most tasks that complete successfully should return Normal.
	Normal TaskRetVal = iota
	// Fail indicates requested authorization or elevation failed, or a Go task
	// or Job failed or ended unsucesfully.
	Fail
	// MechanismFail indicates authorization or elevation couldn't be determined
	// due to a technical issue that should be logged. Should be returned by
	// tasks, jobs and plugins for unusual errors, akin to http 500.
	MechanismFail
	// ConfigurationError indicates authorization or elevation failed due to misconfiguration
	ConfigurationError
	// PipelineAborted - failed exclusive w/o queueTask
	PipelineAborted
	// RobotStopping - the robot is shutting down and can't start any new pipelines
	RobotStopping
	// NotFound - generic return value when the asked for item couldn't be returned
	NotFound
	// Success indicates successful authorization or elevation; using '7' (three bits set)
	// reduces the likelihood of an authorization plugin mistakenly exiting with a success
	// value
	Success = 7
)

// RetVal is a integer type for returning error conditions from bot methods, or 0 for Ok
type RetVal int

const (
	// Ok indicates a successful result
	Ok RetVal = iota // success

	/* Connector issues */

	// UserNotFound - failed lookup
	UserNotFound
	// ChannelNotFound - failed lookup
	ChannelNotFound
	// AttributeNotFound - failed looking up user/robot attributes like email, name, etc.
	AttributeNotFound
	// FailedMessageSend - the bot was not able to send a message
	FailedMessageSend
	// FailedChannelJoin - the robot couldn't join a channel; e.g. slack doesn't allow bots to join
	FailedChannelJoin

	/* Brain Maladies */

	// DatumNotFound - key not found in the global hash when update called
	DatumNotFound
	// DatumLockExpired - A datum was checked out for too long, and the lock expired
	DatumLockExpired
	// DataFormatError - Problem unmarshalling JSON
	DataFormatError
	// BrainFailed - An error condition prevented the brain from storing/retrieving; redis down, file write failed, etc.
	BrainFailed
	// InvalidDatumKey - Key name didn't match the regex for valid key names
	InvalidDatumKey

	/* GetTaskConfig */

	// InvalidConfigPointer - GetTaskConfig requires a pointer to a config struct
	InvalidConfigPointer
	// ConfigUnmarshalError - Unmarshalling failed
	ConfigUnmarshalError
	// NoConfigFound - The plugin/job doesn't have any config data
	NoConfigFound

	/* Prompt(User)ForReply */

	// RetryPrompt - There was already a prompt in progress for the user/channel
	RetryPrompt
	// ReplyNotMatched - The user reply didn't match the pattern waited for
	ReplyNotMatched
	// UseDefaultValue - The user replied with a single '=', meaning use a default value
	UseDefaultValue
	// TimeoutExpired - The user didn't reply within the given timeout
	TimeoutExpired
	// Interrupted - The user replied with '-' (cancel)
	Interrupted
	// MatcherNotFound - There was no matcher configured with the given string, or the regex didn't compile
	MatcherNotFound

	/* Email */

	// NoUserEmail - Couldn't look up the user's email address
	NoUserEmail
	// NoBotEmail - Couldn't look up the robot's email address
	NoBotEmail
	// MailError - There was an error sending email
	MailError

	/* Pipeline errors */

	// TaskNotFound - no task with the given name
	TaskNotFound
	// MissingArguments - AddTask requires a command and args for a plugin
	MissingArguments
	// InvalidStage - tasks can only be added when the robot is running primaryTasks
	InvalidStage
	// InvalidTaskType - mismatch of task/plugin/job method with provided name
	InvalidTaskType
	// CommandNotMatched - the command string didn't match a command for the plugin
	CommandNotMatched
	// TaskDisabled - a method call attempted to add a disabled task to a pipeline
	TaskDisabled
	// PrivilegeViolation - error adding a privileged job/command to an unprivileged pipeline
	PrivilegeViolation
	// Failed is a generic failure code for use when we don't want to return Ok;
	// should be accompanied by a log.
	Failed = 63
)

// MessageFormat indicates how the connector should display the content of
// the message. One of Variable, Fixed or Raw
type MessageFormat int

// Outgoing message format, Variable or Fixed
const (
	Raw MessageFormat = iota // protocol native, zero value -> default if not specified
	Fixed
	Variable
)
