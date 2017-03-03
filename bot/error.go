package bot

// RetVal is a bit field for returning error conditions, or 0 for Ok
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
	// FailedUserDM - the bot was not able to send an direct message (DM) to a given user
	FailedUserDM
	// FailedChannelJoin - the robot couldn't join a channel; e.g. slack doesn't allow bots to join
	FailedChannelJoin

	/* Brain Maladies */

	// DatumNotFound - name not found in the global table when update called
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

	/* OTP */

	// NoUserOTP - OTP config for the user wasn't found
	NoUserOTP
	// OTPError - error value returned while checking OTP
	OTPError

	/* WaitForReply */

	// ReplyNotMatched - The user reply didn't match the pattern waited for
	ReplyNotMatched
	// UseDefaultValue - The user replied with a single '=', meaning use a default value
	UseDefaultValue
	// TimeoutExpired - The user didn't reply within the given timeout
	TimeoutExpired
	// ReplyInProgress - The robot is already waiting for a reply from the user in a given channel
	ReplyInProgress
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
