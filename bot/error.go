package bot

// BotRetVal is a bit field for returning error conditions, or 0 for Ok
type BotRetVal int

const (
	Ok BotRetVal = iota // success

	// Connector issues
	UserNotFound      // obvious
	ChannelNotFound   // obvious
	AttributeNotFound // Looking up user/robot attributes like email, name, etc.
	FailedUserDM      // Not able to send an direct message (DM) to a given user
	FailedChannelJoin // Not able to join a channel; e.g. slack doesn't allow bots to join

	// Brain Maladies
	DatumNotFound    // Datum name not found in the global table when update called
	DatumLockExpired // A datum was checked out for too long, and the lock expired
	DataFormatError  // Problem unmarshalling JSON
	BrainFailed      // An error condition prevented the brain from storing/retrieving; redis down, file write failed, etc.
	InvalidDatumKey  // Key name didn't match the regex for valid key names

	// GetPluginConfig
	InvalidDblPtr    // GetPluginConfig wasn't called with a double-pointer to a config struct
	InvalidCfgStruct // The struct type in GetPluginConfig doesn't match the struct registered for the plugin

	// OTP
	UntrustedPlugin // A plugin called CheckOTP without having Trusted: true configured
	NoUserOTP       // OTP config for the user wasn't found
	OTPError        // Error value returned while checking OTP

	// WaitForReply
	ReplyNotMatched // The user reply didn't match the pattern waited for
	TimeoutExpired  // The user didn't reply within the given timeout
	ReplyInProgress // The robot is already waiting for a reply from the user in a given channel
	MatcherNotFound // There was no matcher configured with the given string

	// Email
	NoUserEmail // Couldn't look up the user's email address
	NoBotEmail  // Couldn't look up the robot's email address
	MailError   // There was an error sending email
)

func (ret BotRetVal) String() string {
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
		"Plugin requested OTP but not configured by administrator as Trusted",
		"User OTP configuration not found",
		"Unspecified error in checking OTP",
		"The user's reply didn't match the requested regex",
		"The user didn't respond within the given timeout",
		"The user is already engaged in an interactive command with the robot in the same channel",
		"The matcher key supplied in WaitForReply doesn't correspond to a configured regex",
		"User email attribute not available",
		"Robot email attribute not available",
		"Unspecified error sending email",
	}
	return errMsg[int(ret)]
}
