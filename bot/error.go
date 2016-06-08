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
