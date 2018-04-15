package bot

// RetVal is a integer type for returning error conditions from bot methods, or 0 for Ok
type RetVal int

//go:generate stringer -type=PlugRetVal

// PlugRetVal is an integer type for return values from plugins, mainly for elevation & authorization
type PlugRetVal int

const (
	// Normal exit is for non-auth/non-elevating plugins; since this is the
	// default exit value, we don't use it to indicate successful authentication
	// or elevation.
	Normal PlugRetVal = iota
	// Fail indicates requested authorization or elevation failed
	Fail
	// MechanismFail indicates authorization or elevation couldn't be determined due to a technical issue that should be logged
	MechanismFail
	// ConfigurationError indicates authorization or elevation failed due to misconfiguration
	ConfigurationError
	// UntrustedPlugin indicates a plugin was called by an untrusted plugin
	UntrustedPlugin
	// Success indicates successful authorization or elevation; using '7' (three bits set)
	// reduces the likelihood of an authorization plugin mistakenly exiting with a success
	// value
	Success = 7
)

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
	// FailedUserDM - the bot was not able to send an direct message (DM) to a given user
	FailedUserDM
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

	/* GetPluginConfig */

	// InvalidDblPtr - GetPluginConfig wasn't called with a double-pointer to a config struct
	InvalidDblPtr
	// InvalidCfgStruct - The struct type in GetPluginConfig doesn't match the struct registered for the plugin
	InvalidCfgStruct
	// NoConfigFound - The plugin doesn't have any config data
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
	// Interrupted - The user issued another command instead of replying, or replied with '-' (cancel)
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
)

func (ret RetVal) String() string {
	errMsg := []string{
		"Ok",
		"User not found",
		"Channel not found",
		"Attribute not found",
		"Failed sending direct message to user",
		"Failed to join the channel",
		"Datum not found in robot's brain",
		"Datum lock expired before update",
		"Problem unmarshalling JSON/Yaml",
		"Brain storage failed",
		"Invalid string given for datum key",
		"Argument to GetPluginConfig wasn't a double pointer to a struct",
		"Mismatch between struct registered in RegisterPlugin and struct passed to GetPluginConfig",
		"Plugin configuration didn't have a Config section",
		"User OTP configuration not found",
		"Unspecified error in checking OTP",
		"The user's reply didn't match the requested regex",
		"The user didn't respond within the given timeout",
		"The user is already engaged in an interactive command with the robot in the same channel",
		"The matcher key supplied in WaitForReply doesn't correspond to a configured regex, or a provided regex didn't compile",
		"User email attribute not available",
		"Robot email attribute not available",
		"Unspecified error sending email",
	}
	return errMsg[int(ret)]
}
