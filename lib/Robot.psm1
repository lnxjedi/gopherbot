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

function enc64([String] $msg) {
    [OutputType([String])]
    $b  = [System.Text.Encoding]::UTF8.GetBytes($msg)
    $enc = [System.Convert]::ToBase64String($b)
    return "base64:$enc"
}

function dec64([String] $msg) {
    [OutputType([String])]
    [String[]] $parts = $msg.Split(":")
    if ($parts[0] -eq "base64") {
        $b  = [System.Convert]::FromBase64String($parts[1])
        return [System.Text.Encoding]::UTF8.GetString($b)
    } else {
        return $msg
    }
}

class Attribute {
    # Properties
    [String] $Attr
    [BotRet] $Ret

    Attribute([String] $Attr, [BotRet] $Ret) {
        $this.Attr = $Attr
        $this.Ret = $Ret
    }

    [String] ToString() {
        return $this.Attr
    }
}

class Reply {
    [String] $Reply
    [BotRet] $Ret

    Attribute([String] $rep, [BotRet] $ret) {
        $this.Reply = $rep
        $this.Ret = $ret
    }

    [String] ToString() {
        return $this.Reply
    }
}

class OTPRet {
    [Bool] $Valid
    [BotRet] $Ret

    OTPRet([Bool] $v, [BotRet] $r) {
        $this.Valid = $v
        $this.Ret = $r
    }
}

class BotFuncCall {
    [String] $FuncName
    [String] $User
    [String] $Channel
    [String] $Format
    [String] $PluginID
    [PSCustomObject] $FuncArgs

    BotFuncCall([String] $fn, [String] $u, [String] $c, [String] $fmt, [String] $p, [PSCustomObject] $funcArgs ) {
        $this.FuncName = $fn
        $this.User = $u
        $this.Channel = $c
        $this.Format = $fmt
        $this.PluginID = $p
        $this.FuncArgs = $funcArgs
        $aj = $funcArgs | ConvertTo-Json
    }
}

class Robot
{
    # Properties
    [String] $Channel
    [String] $User
    hidden [String] $PluginID

    # Constructor
    Robot([String] $Channel, [String] $User, [String] $PluginID) {
        $this.Channel = $Env:GOPHER_CHANNEL
        $this.User = $Env:GOPHER_USER
        $this.PluginID = $Env:GOPHER_PLUGIN_ID
    }

    [Robot] Direct() {
        return [Robot]::new("", $this.User, $this.PluginID)
    }

    [PSCustomObject] Call([String] $fname, [PSCustomObject] $funcArgs, [String] $format) {
        $fc = [BotFuncCall]::new($fname, $this.User, $this.Channel, $format, $this.PluginID, $funcArgs) | ConvertTo-Json
        if ($fname -ne "Log") { $this.Log("Debug", "DEBUG - Sending: $fc") }
        $r = Invoke-WebRequest -URI "$Env:GOPHER_HTTP_POST/json" -Method Post -Body $fc
        $c = $r.Content
        if ($fname -ne "Log") { $this.Log("Debug", "DEBUG - Got back: $c") }
        return $r.Content | ConvertFrom-Json
    }

    [PSCustomObject] Call([String] $fname, [PSCustomObject] $funcArgs) {
        return $this.Call($fname, $funcArgs, "variable")
    }

    [Attribute] GetSenderAttribute([String] $attr) {
        $funcArgs = [PSCustomObject]@{ Attribute=$attr }
        $ret = $this.Call("GetSenderAttribute", $funcArgs)
        $a = dec64($ret.Attribute)
        return [Attribute]::new($a, $ret.RetVal -As [BotRet])
    }

    [Attribute] GetUserAttribute([String] $user, [String] $attr) {
        $funcArgs = [PSCustomObject]@{ User=$user; Attribute=$attr }
        $ret = $this.Call("GetUserAttribute", $funcArgs)
        $a = dec64($ret.Attribute)
        return [Attribute]::new($a, $ret.RetVal -As [BotRet])
    }

    [Attribute] GetBotAttribute([String] $attr) {
        $funcArgs = [PSCustomObject]@{ Attribute=$attr }
        $ret = $this.Call("GetBotAttribute", $funcArgs)
        $a = dec64($ret.Attribute)
        return [Attribute]::new($a, $ret.RetVal -As [BotRet])
    }

    [Reply] WaitForReply([String] $regexid, [Int] $timeout) {
        $funcArgs = [PSCustomObject]@{ RegExId=$regexid; Timeout=$timeout }
        $ret = $this.Call("WaitForReply", $funcArgs)
        $rep = dec64($ret.Reply)
        return [Reply]::new($rep, $ret.Ret -As [BotRet])
    }

    [Reply] WaitForReply([String] $regexid) {
        return $this.WaitForReply($regexid, 30)
    }

    [Reply] WaitForReplyRegex([String] $goregex, [Int] $timeout) {
        $funcArgs = [PSCustomObject]@{ RegEx=$goregex; Timeout=$timeout }
        $ret = $this.Call("WaitForReply", $funcArgs)
        $rep = dec64($ret.Reply)
        return [Reply]::new($rep, $ret.Ret -As [BotRet])
    }

    [Reply] WaitForReplyRegex([String] $goregex) {
        return $this.WaitForReplyRegex($goregex, 30)
    }

    Log([String] $level, [String] $message) {
        $funcArgs = [PSCustomObject]@{ Level=$level; Message=$message }
        $this.Call("Log", $funcArgs)
    }

    [BotRet] SendChannelMessage([String] $channel, [String] $msg, [String] $format="variable") {
        $funcArgs = [PSCustomObject]@{ Channel=$channel; Message=enc64($msg) }
        return $this.Call("SendChannelMessage", $funcArgs, $format).RetVal -As [BotRet]
    }

    [BotRet] SendChannelMessage([String] $channel, [String] $msg) {
        return $this.SendChannelMessage($channel, $msg, "variable")
    }

    [BotRet] SendUserMessage([String] $user, [String] $msg, [String] $format="variable") {
        $funcArgs = [PSCustomObject]@{ User=$user; Message=enc64($msg) }
        return $this.Call("SendUserMessage", $funcArgs, $format).RetVal -As [BotRet]
    }

    [BotRet] SendUserMessage([String] $user, [String] $msg) {
        return $this.SendUserMessage($user, $msg, "variable")
    }

    [BotRet] SendUserChannelMessage([String] $user, [String] $channel, [String] $msg, [String] $format="variable") {
        $funcArgs = [PSCustomObject]@{ User=$user; Channel=$channel; Message=enc64($msg) }
        return $this.Call("SendUserChannelMessage", $funcArgs, $format).RetVal -As [BotRet]
    }

    [BotRet] SendUserChannelMessage([String] $user, [String] $channel, [String] $msg) {
        return $this.SendUserChannelMessage($user, $channel, $msg, "variable")
    }

    [BotRet] Say([String] $msg, [String] $format) {
        if ($this.Channel -eq ""){
            return $this.SendUserMessage($this.User, $msg, $format)
        } else {
            return $this.SendChannelMessage($this.Channel, $msg, $format)
        }
    }

    [BotRet] Say([String] $msg) {
        return $this.Say($msg, "variable")
    }

    [BotRet] Reply([String] $msg, [String] $format = "variable") {
        if ($this.Channel -eq "") {
            return $this.SendUserMessage($this.User, $msg, $format)
        } else {
            return $this.SendUserChannelMessage($this.User, $this.Channel, $msg, $format)
        }
    }

    [BotRet] Reply([String] $msg) {
        return $this.Reply($msg, "variable")
    }
}

function Get-Robot([String] $Channel, [String] $User, [String] $PluginID) {
    return [Robot]::new($Channel, $User, $PluginID)
}

export-modulemember -function Get-Robot