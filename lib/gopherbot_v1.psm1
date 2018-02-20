# Return values for robot method calls
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
    RetryPrompt = 14
    ReplyNotMatched = 15
    UseDefaultValue = 16
    TimeoutExpired = 17
    Interrupted = 18
    MatcherNotFound = 19
    NoUserEmail = 20
    NoBotEmail = 21
    MailError = 22
}

# Plugin return values / exit codes, return values from CallPlugin
Enum PlugRet
{
    Normal = 0
    Success = 1
    Fail = 2
    MechanismFail = 3
    ConfigurationError = 4
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

    Attribute([PSCustomObject] $obj) {
        $a = dec64($obj.Attribute)
        $this.Attr = $a
        $this.Ret = $obj.RetVal -As [BotRet]
    }

    [String] ToString() {
        return $this.Attr
    }
}

class Reply {
    [String] $Reply
    [BotRet] $Ret

    Reply([String] $rep, [Int] $ret) {
        $this.Reply = $rep
        $this.Ret = $ret -As [BotRet]
    }

    [String] ToString() {
        return $this.Reply
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
    }
}

class Robot
{
    # Properties
    [String] $Channel
    [String] $User
    hidden [String] $PluginID

    # Constructor
    Robot([String] $channel, [String] $user, [String] $pluginid) {
        $this.Channel = $channel
        $this.User = $user
        $this.PluginID = $pluginid
    }

    [Robot] Direct() {
        return [Robot]::new("", $this.User, $this.PluginID)
    }

    Pause([single] $seconds) {
        Start-Sleep $seconds
    }

    [bool] CheckAdmin() {
        return $this.Call("CheckAdmin", $null).Boolean -As [bool]
    }

    [bool] Elevate([bool] $immediate) {
        $funcArgs = [PSCustomObject]@{ Immediate=$immediate }
        return $this.Call("Elevate", $funcArgs).Boolean -As [bool]
    }

    [bool] Elevate() {
        return $this.Elevate($FALSE)
    }

    [int] RandomInt([int] $i) {
        return Get-Random -Maximum $i
    }

    [string] RandomString([String[]] $sarr) {
        $l = $sarr.Count
        $i = Get-Random -Maximum $l
        return $sarr[$i]
    }

    [PSCustomObject] GetPluginConfig() {
        return $this.Call("GetPluginConfig", $null)
    }

    [PSCustomObject] Call([String] $fname, [PSCustomObject] $funcArgs, [String] $format) {
        $bfc = [BotFuncCall]::new($fname, $this.User, $this.Channel, $format, $this.PluginID, $funcArgs)
        $fc = ConvertTo-Json $bfc
        # if ($fname -ne "Log") { $this.Log("Debug", "DEBUG - Sending: $fc") }
        $r = Invoke-WebRequest -URI "$Env:GOPHER_HTTP_POST/json" -Method Post -UseBasicParsing -Body $fc
        $c = $r.Content
        # if ($fname -ne "Log") { $this.Log("Debug", "DEBUG - Got back: $c") }
        return ConvertFrom-Json $c
    }

    [PSCustomObject] Call([String] $fname, [PSCustomObject] $funcArgs) {
        return $this.Call($fname, $funcArgs, "variable")
    }

    [PlugRet] CallPlugin([String] $plugName, [String[]]$plugArgs) {
        $funcArgs = [PSCustomObject]@{ PluginName=$plugName }
        $ret = $this.Call("CallPlugin", $funcArgs)
        if ([PlugRet]$ret.PlugRetVal -ne "Success") {
            return [PlugRet]$ret.PlugRetVal
        }
        if ( $ret.InterpreterPath -match "powershell" ) {
            $plugPath = $ret.PluginPath -replace ' ','` '
        } else {
            $plugPath = $ret.PluginPath
        }
        $plugArgs = [Array]$plugPath + $plugArgs
        $Env:GOPHER_PLUGIN_ID = $ret.PluginID
        $proc = Start-Process -FilePath $ret.InterpreterPath -ArgumentList $plugArgs -NoNewWindow -PassThru -Wait
        $Env:GOPHER_PLUGIN_ID = $this.PluginID
        return [PlugRet]$proc.ExitCode
    }

    [PSCustomObject] CheckoutDatum([String] $key, [Bool] $rw) {
        $funcArgs = [PSCustomObject]@{ Key=$key; RW=$rw }
        $ret = $this.Call("CheckoutDatum", $funcArgs)
        $ret | Add-Member -NotePropertyName Key -NotePropertyValue $key
        return $ret
    }

    CheckinDatum([PSCustomObject] $mem){
        $funcArgs = [PSCustomObject]@{ Key=$mem.Key; Token=$mem.LockToken }
        $this.Call("CheckinDatum", $funcArgs)
    }

    [BotRet] UpdateDatum([PSCustomObject] $mem){
        $funcArgs = [PSCustomObject]@{ Key=$mem.Key; Token=$mem.LockToken; Datum=$mem.Datum }
        $ret = $this.Call("UpdateDatum", $funcArgs)
        return $ret.RetVal -As [BotRet]
    }

    [BotRet] Remember([String] $key, [String] $value){
        $funcArgs = [PSCustomObject]@{ Key=$key; Value=$value }
        $ret = $this.Call("Remember", $funcArgs)
        return $ret.RetVal -As [BotRet]
    }

    [BotRet] RememberContext([String] $context, [String] $value){
        return $this.Remember("context:"+$context, $value)
    }

    [String] Recall([String] $key){
        $funcArgs = [PSCustomObject]@{ Key=$key }
        $ret = $this.Call("Recall", $funcArgs)
        return $ret.StrVal
    }

    [Attribute] GetSenderAttribute([String] $attr) {
        $funcArgs = [PSCustomObject]@{ Attribute=$attr }
        $ret = $this.Call("GetSenderAttribute", $funcArgs)
        return [Attribute]::new($ret)
    }

    [Attribute] GetUserAttribute([String] $user, [String] $attr) {
        $funcArgs = [PSCustomObject]@{ User=$user; Attribute=$attr }
        $ret = $this.Call("GetUserAttribute", $funcArgs)
        return [Attribute]::new($ret)
    }

    [Attribute] GetBotAttribute([String] $attr) {
        $funcArgs = [PSCustomObject]@{ Attribute=$attr }
        $ret = $this.Call("GetBotAttribute", $funcArgs)
        return [Attribute]::new($ret)
    }

    [Reply] PromptForReply([String] $regexid, [String] $prompt) {
        return $this.PromptUserChannelForReply($regexid, $this.User, $this.Channel, $prompt)
    }

    [Reply] PromptUserForReply([String] $regexid, [String] $user, [String] $prompt) {
        return $this.PromptUserChannelForReply($regexid, $user, "", $prompt)
    }

    [Reply] PromptUserChannelForReply([String] $regexid, [String] $user, [String] $channel, [String] $prompt) {
        $funcArgs = [PSCustomObject]@{ RegexID=$regexid; User=$user; Channel=$channel; Prompt=$prompt }
        $ret = $null
        For ($i=0; $i -le 3; $i++) {
            $ret = $this.Call("PromptUserChannelForReply", $funcArgs)
            if ([int]$ret.RetVal -eq "RetryPrompt" ){ continue }
            $rep = dec64($ret.Reply)
            return [Reply]::new($rep, $ret.Ret -As [BotRet])
        }
        $rep = dec64($ret.Reply)
        if ($ret -eq "RetryPrompt" ) {
            return [Reply]::new($rep, [BotRet]("Interrupted"))
        }
        return [Reply]::new($rep.Reply, $ret.RetVal -As [BotRet])
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

function Get-Robot() {
    return [Robot]::new($Env:GOPHER_CHANNEL, $Env:GOPHER_USER, $Env:GOPHER_PLUGIN_ID)
}

export-modulemember -function Get-Robot
# SIG # Begin signature block
# MIIOSwYJKoZIhvcNAQcCoIIOPDCCDjgCAQExCzAJBgUrDgMCGgUAMGkGCisGAQQB
# gjcCAQSgWzBZMDQGCisGAQQBgjcCAR4wJgIDAQAABBAfzDtgWUsITrck0sYpfvNR
# AgEAAgEAAgEAAgEAAgEAMCEwCQYFKw4DAhoFAAQUGQV1POAW5lfYDuhIRsr4BiJu
# WWCggguCMIIFjzCCBHegAwIBAgIRAJJHZXGVpHVKJI9gIzdPrk8wDQYJKoZIhvcN
# AQELBQAwfDELMAkGA1UEBhMCVVMxCzAJBgNVBAgTAk1JMRIwEAYDVQQHEwlBbm4g
# QXJib3IxEjAQBgNVBAoTCUludGVybmV0MjERMA8GA1UECxMISW5Db21tb24xJTAj
# BgNVBAMTHEluQ29tbW9uIFJTQSBDb2RlIFNpZ25pbmcgQ0EwHhcNMTgwMjEyMDAw
# MDAwWhcNMjEwMjExMjM1OTU5WjCBqTELMAkGA1UEBhMCVVMxDjAMBgNVBBEMBTIy
# OTA0MQswCQYDVQQIDAJWQTEYMBYGA1UEBwwPQ2hhcmxvdHRlc3ZpbGxlMSEwHwYD
# VQQJDBgyMDE1IEl2eSBSb2FkLCBTdWl0ZSAxMTYxHzAdBgNVBAoMFlVuaXZlcnNp
# dHkgb2YgVmlyZ2luaWExHzAdBgNVBAMMFlVuaXZlcnNpdHkgb2YgVmlyZ2luaWEw
# ggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDNNZbmB4dWL2KA0lfedluO
# Rz/fvXvkui84uO9deIhcBGGlC5QweGNLt/hm2r20fhh77r/Za5Md6wyP3szWGxCc
# hPXUtKfs0rTHWKsSSHQpW0uD8KVdSdTpqADCi6qzQarqS1CRrS1j+pL4KHK0v8ly
# yIo3cGAsyswR2narPfFfvaz0CcQ/YsO9JhsbZGlXPSXEsMgvMdDpu+9ycSGQtRel
# 6YeqrKnFFTqjhiDaLH2WWyBvrSh69mV2aoRvzsXeDhGMYB0Tpv0Rpbqg3nPCvLgF
# kiSMF+IDgkW+MZv6MRQXxpl8hvcJ7RaoeDlYkcWbu7n1uyDCqO2FF2XfUbDpvLVv
# AgMBAAGjggHcMIIB2DAfBgNVHSMEGDAWgBSuNSMX//8GPZxQ4IwkZTMecBCIojAd
# BgNVHQ4EFgQUhqM+wzTdA4Lfg7KccGYc2zu4RKcwDgYDVR0PAQH/BAQDAgeAMAwG
# A1UdEwEB/wQCMAAwEwYDVR0lBAwwCgYIKwYBBQUHAwMwEQYJYIZIAYb4QgEBBAQD
# AgQQMGYGA1UdIARfMF0wWwYMKwYBBAGuIwEEAwIBMEswSQYIKwYBBQUHAgEWPWh0
# dHBzOi8vd3d3LmluY29tbW9uLm9yZy9jZXJ0L3JlcG9zaXRvcnkvY3BzX2NvZGVf
# c2lnbmluZy5wZGYwSQYDVR0fBEIwQDA+oDygOoY4aHR0cDovL2NybC5pbmNvbW1v
# bi1yc2Eub3JnL0luQ29tbW9uUlNBQ29kZVNpZ25pbmdDQS5jcmwwfgYIKwYBBQUH
# AQEEcjBwMEQGCCsGAQUFBzAChjhodHRwOi8vY3J0LmluY29tbW9uLXJzYS5vcmcv
# SW5Db21tb25SU0FDb2RlU2lnbmluZ0NBLmNydDAoBggrBgEFBQcwAYYcaHR0cDov
# L29jc3AuaW5jb21tb24tcnNhLm9yZzAdBgNVHREEFjAUgRJkbHA3eUB2aXJnaW5p
# YS5lZHUwDQYJKoZIhvcNAQELBQADggEBAKdp38HN09Hu5BNhbbbcmOrimPhHEd5b
# r7gq94i/VS4sAEspUCpR4LH0JcZKICvbmJvKuLGZn1I/viE7KZ025viumXVu65mf
# 8fRv3HHsLvNmFGtVXA85BQerLMnHZ+cQ172c1/kXaWNAP/PwlkWGs/jR8Md2J8mo
# kpGMBz7E5+jT6lh8T3Qp4DwGLXUV7bnHJs5Ww6RyMtBd6iRY5kUWv/xE9JILwSwO
# mbf4Y/6ov75DAJpXUs1owwAJtT9Hr/SYW95e1wxOqrENDReSTOfY9uNhmsq1nY77
# /0otg7JBGY2CAkaEmIyPUB05S5LLN+eHKLMsaFjoGfe9iJ4NeicFrRwwggXrMIID
# 06ADAgECAhBl4eLj1d5QRYXzJiSABeLUMA0GCSqGSIb3DQEBDQUAMIGIMQswCQYD
# VQQGEwJVUzETMBEGA1UECBMKTmV3IEplcnNleTEUMBIGA1UEBxMLSmVyc2V5IENp
# dHkxHjAcBgNVBAoTFVRoZSBVU0VSVFJVU1QgTmV0d29yazEuMCwGA1UEAxMlVVNF
# UlRydXN0IFJTQSBDZXJ0aWZpY2F0aW9uIEF1dGhvcml0eTAeFw0xNDA5MTkwMDAw
# MDBaFw0yNDA5MTgyMzU5NTlaMHwxCzAJBgNVBAYTAlVTMQswCQYDVQQIEwJNSTES
# MBAGA1UEBxMJQW5uIEFyYm9yMRIwEAYDVQQKEwlJbnRlcm5ldDIxETAPBgNVBAsT
# CEluQ29tbW9uMSUwIwYDVQQDExxJbkNvbW1vbiBSU0EgQ29kZSBTaWduaW5nIENB
# MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAwKAvix56u2p1rPg+3KO6
# OSLK86N25L99MCfmutOYMlYjXAaGlw2A6O2igTXrC/Zefqk+aHP9ndRnec6q6mi3
# GdscdjpZh11emcehsriphHMMzKuHRhxqx+85Jb6n3dosNXA2HSIuIDvd4xwOPzSf
# 5X3+VYBbBnyCV4RV8zj78gw2qblessWBRyN9EoGgwAEoPgP5OJejrQLyAmj91QGr
# 9dVRTVDTFyJG5XMY4DrkN3dRyJ59UopPgNwmucBMyvxR+hAJEXpXKnPE4CEqbMJU
# vRw+g/hbqSzx+tt4z9mJmm2j/w2nP35MViPWCb7hpR2LB8W/499Yqu+kr4LLBfgK
# CQIDAQABo4IBWjCCAVYwHwYDVR0jBBgwFoAUU3m/WqorSs9UgOHYm8Cd8rIDZssw
# HQYDVR0OBBYEFK41Ixf//wY9nFDgjCRlMx5wEIiiMA4GA1UdDwEB/wQEAwIBhjAS
# BgNVHRMBAf8ECDAGAQH/AgEAMBMGA1UdJQQMMAoGCCsGAQUFBwMDMBEGA1UdIAQK
# MAgwBgYEVR0gADBQBgNVHR8ESTBHMEWgQ6BBhj9odHRwOi8vY3JsLnVzZXJ0cnVz
# dC5jb20vVVNFUlRydXN0UlNBQ2VydGlmaWNhdGlvbkF1dGhvcml0eS5jcmwwdgYI
# KwYBBQUHAQEEajBoMD8GCCsGAQUFBzAChjNodHRwOi8vY3J0LnVzZXJ0cnVzdC5j
# b20vVVNFUlRydXN0UlNBQWRkVHJ1c3RDQS5jcnQwJQYIKwYBBQUHMAGGGWh0dHA6
# Ly9vY3NwLnVzZXJ0cnVzdC5jb20wDQYJKoZIhvcNAQENBQADggIBAEYstn9qTiVm
# vZxqpqrQnr0Prk41/PA4J8HHnQTJgjTbhuET98GWjTBEE9I17Xn3V1yTphJXbat5
# l8EmZN/JXMvDNqJtkyOh26owAmvquMCF1pKiQWyuDDllxR9MECp6xF4wnH1Mcs4W
# eLOrQPy+C5kWE5gg/7K6c9G1VNwLkl/po9ORPljxKKeFhPg9+Ti3JzHIxW7Ldylj
# ffccWiuNFR51/BJHAZIqUDw3LsrdYWzgg4x06tgMvOEf0nITelpFTxqVvMtJhnOf
# ZbpdXZQ5o1TspxfTEVOQAsp05HUNCXyhznlVLr0JaNkM7edgk59zmdTbSGdMq8Zt
# uu6VyrivOlMSPWmay5MjvwTzuNorbwBv0DL+7cyZBp7NYZou+DoGd1lFZN0jU5Is
# QKgm3+00pnnJ67crdFwfz/8bq3MhTiKOWEb04FT3OZVp+jzvaChHWLQ8gbCORgCl
# aZq1H3aqI7JeRkWEEEp6Tv4WAVsr/i7LoXU72gOb8CAzPFqwI4Excdrxp0I4OXbE
# CHlDqU4sTInqwlMwofmxeO4u94196qIqJQl+8Sykl06VktqMux84Iw3ZQLH08J8L
# aJ+WDUycc4OjY61I7FGxCDkbSQf3npXeRFm0IBn8GiW+TRDk6J2XJFLWEtVZmhbo
# FlBLoUlqHUCKu0QOhU/+AEOqnY98j2zRMYICMzCCAi8CAQEwgZEwfDELMAkGA1UE
# BhMCVVMxCzAJBgNVBAgTAk1JMRIwEAYDVQQHEwlBbm4gQXJib3IxEjAQBgNVBAoT
# CUludGVybmV0MjERMA8GA1UECxMISW5Db21tb24xJTAjBgNVBAMTHEluQ29tbW9u
# IFJTQSBDb2RlIFNpZ25pbmcgQ0ECEQCSR2VxlaR1SiSPYCM3T65PMAkGBSsOAwIa
# BQCgeDAYBgorBgEEAYI3AgEMMQowCKACgAChAoAAMBkGCSqGSIb3DQEJAzEMBgor
# BgEEAYI3AgEEMBwGCisGAQQBgjcCAQsxDjAMBgorBgEEAYI3AgEVMCMGCSqGSIb3
# DQEJBDEWBBSLIW8Qb3/9UbCuj6xh7Q9qKZTDXjANBgkqhkiG9w0BAQEFAASCAQAs
# bbwhPkcy3ifKyfzAQAqVYFYVLNGzk1MO1rjr2wJrhNmnMSnrSGWYn0k2Cn9FeLG0
# pIf2/RboXo6xO+XjAvYne3htZuCPipOuqKuNRHxZst32XCP1QQON0pFgwTT6DZuo
# iTuQJjyFSmLCm1nsfdP/XRoygotFV8obBDCKbSYaXBhY3HXyMD6FTBJAkyVSM4Lu
# VfeDx2Oma+ucMY/p12fd8ZCbFEvHfZg/ES3mI0WRmnDjQM5IOJD+Zft/lJeXzYMS
# WS0GuAA1SZ18avUPCxR97sv4SJa2bU04MpEtOPpx84rjHYHXMqjJrHPpBilF+tzu
# TseLmBUgsqt/OSBu4yN0
# SIG # End signature block
