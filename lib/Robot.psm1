Enum BotRet
{
    Ok = 0
	UserNotFound = 1
	ChannelNotFound = 2
	AttributeNotFound = 3
	FailedUserDM = 4
	FailedChannelJoin = 5
	DatumNotFound = 6
	DatumLockExpired = 7
	DataFormatError = 8
	BrainFailed = 9
	InvalidDatumKey = 10
	InvalidDblPtr = 11
	InvalidCfgStruct = 12
	NoConfigFound = 13
	NoUserOTP = 14
	OTPError = 15
	ReplyNotMatched = 16
	UseDefaultValue = 17
	TimeoutExpired = 18
	Interrupted = 19
	MatcherNotFound = 20
	NoUserEmail = 21
	NoBotEmail = 22
	MailError = 23
}

class Attribute {
    # Peroperties
    [String] $Attr
    [BotRet] $Ret

    Attribute([String] $Attr, [BotRet] $Ret)
    {
        $this.Attr = $Attr
        $this.Ret = $Ret
    }
}

class Robot
{
    # Properties
    [String] $Channel
    [String] $User
    hidden [String] $PluginID

    # Constructor
    Robot([String] $Channel, [String] $User, [String] $PluginID)
    {
        $this.Channel = $Channel
        $this.User = $User
        $this.PluginID = $PluginID
    }

    [Robot] Direct()
    {
        return [Robot]::new("", $this.User, $this.PluginID)
    }
}

function Get-Robot([String] $Channel, [String] $User, [String] $PluginID) {
    return [Robot]::new($channel, $user, $plugID)
}

export-modulemember -function Get-Robot