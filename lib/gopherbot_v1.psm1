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

# Plugin return values / exit codes
Enum PlugRet
{
    Normal = 0
    Fail = 1
    MechanismFail = 2
    ConfigurationError = 3
    Success = 7
}

class Attribute {
    # Properties
    [String] $Attr
    [BotRet] $Ret

    Attribute([PSCustomObject] $obj) {
        $this.Attr = $obj.Attribute
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
    [String] $Protocol
    [String] $CallerID
    [PSCustomObject] $FuncArgs

    BotFuncCall([String] $fn, [String] $u, [String] $c, [String] $pr, [String] $fmt, [String] $p, [PSCustomObject] $funcArgs ) {
        $this.FuncName = $fn
        $this.User = $u
        $this.Channel = $c
        $this.Protocol = $pr
        $this.Format = $fmt
        $this.CallerID = $p
        $this.FuncArgs = $funcArgs
    }
}

class Robot
{
    # Properties
    [String] $Channel
    [String] $User
    [String] $Protocol
    [String] $Format
    hidden [String] $CallerID

    # Constructor
    Robot([String] $channel, [String] $user, [String] $proto, [String] $format, [String] $pluginid) {
        $this.Channel = $channel
        $this.User = $user
        $this.Protocol = $proto
        $this.Format = $format
        $this.CallerID = $pluginid
    }

    [Robot] Direct() {
        return [Robot]::new("", $this.User, $this.Protocol, $this.Format, $this.CallerID)
    }

    [Robot] MessageFormat([String] $format) {
        return [Robot]::new($this.Channel, $this.User, $this.Protocol, $format, $this.CallerID)
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

    [PSCustomObject] GetTaskConfig() {
        return $this.Call("GetTaskConfig", $null)
    }

    [PSCustomObject] Call([String] $fname, [PSCustomObject] $funcArgs, [String] $format) {
        if ($format.Length -eq 0) {
            $fmt = $this.Format
        } else {
            $fmt = $format
        }
        $bfc = [BotFuncCall]::new($fname, $this.User, $this.Channel, $this.Protocol, $fmt, $this.CallerID, $funcArgs)
        $fc = ConvertTo-Json $bfc
        # if ($fname -ne "Log") { $this.Log("Debug", "DEBUG - Sending: $fc") }
        $r = Invoke-WebRequest -URI "$Env:GOPHER_HTTP_POST/json" -Method Post -UseBasicParsing -Body $fc
        $c = $r.Content
        # if ($fname -ne "Log") { $this.Log("Debug", "DEBUG - Got back: $c") }
        return ConvertFrom-Json $c
    }

    [PSCustomObject] Call([String] $fname, [PSCustomObject] $funcArgs) {
        return $this.Call($fname, $funcArgs, "")
    }
    
    [PlugRet] AddTask([String] $taskName, [String[]]$taskArgs) {
        $funcArgs = [PSCustomObject]@{ Name=$taskName, CmdArgs=$taskArgs }
        $ret = $this.Call("AddTask", $funcArgs)
        return [PlugRet]$ret.PlugRetVal
    }
    
    [Bool] SetParameter([String] $name, [String] $value){
        $funcArgs = [PSCustomObject]@{ Name=$name; Value=$value }
        return $this.Call("SetParameter", $funcArgs).Boolean -As [bool]
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
            $rep = $ret.Reply
            return [Reply]::new($rep, $ret.Ret -As [BotRet])
        }
        $rep = $ret.Reply
        if ($ret -eq "RetryPrompt" ) {
            return [Reply]::new($rep, [BotRet]("Interrupted"))
        }
        return [Reply]::new($rep.Reply, $ret.RetVal -As [BotRet])
    }

    Log([String] $level, [String] $message) {
        $funcArgs = [PSCustomObject]@{ Level=$level; Message=$message }
        $this.Call("Log", $funcArgs)
    }

    [BotRet] SendChannelMessage([String] $channel, [String] $msg, [String] $format) {
        $funcArgs = [PSCustomObject]@{ Channel=$channel; Message=$msg }
        return $this.Call("SendChannelMessage", $funcArgs, $format).RetVal -As [BotRet]
    }

    [BotRet] SendChannelMessage([String] $channel, [String] $msg) {
        return $this.SendChannelMessage($channel, $msg, "")
    }

    [BotRet] SendUserMessage([String] $user, [String] $msg, [String] $format) {
        $funcArgs = [PSCustomObject]@{ User=$user; Message=$msg }
        return $this.Call("SendUserMessage", $funcArgs, $format).RetVal -As [BotRet]
    }

    [BotRet] SendUserMessage([String] $user, [String] $msg) {
        return $this.SendUserMessage($user, $msg, "")
    }

    [BotRet] SendUserChannelMessage([String] $user, [String] $channel, [String] $msg, [String] $format) {
        $funcArgs = [PSCustomObject]@{ User=$user; Channel=$channel; Message=$msg }
        return $this.Call("SendUserChannelMessage", $funcArgs, $format).RetVal -As [BotRet]
    }

    [BotRet] SendUserChannelMessage([String] $user, [String] $channel, [String] $msg) {
        return $this.SendUserChannelMessage($user, $channel, $msg, "")
    }

    [BotRet] Say([String] $msg, [String] $format) {
        if ($this.Channel -eq ""){
            return $this.SendUserMessage($this.User, $msg, $format)
        } else {
            return $this.SendChannelMessage($this.Channel, $msg, $format)
        }
    }

    [BotRet] Say([String] $msg) {
        return $this.Say($msg, "")
    }

    [BotRet] Reply([String] $msg, [String] $format) {
        if ($this.Channel -eq "") {
            return $this.SendUserMessage($this.User, $msg, $format)
        } else {
            return $this.SendUserChannelMessage($this.User, $this.Channel, $msg, $format)
        }
    }

    [BotRet] Reply([String] $msg) {
        return $this.Reply($msg, "")
    }
}

function Get-Robot() {
    return [Robot]::new($Env:GOPHER_CHANNEL, $Env:GOPHER_USER, $Env:GOPHER_PROTOCOL, "", $Env:GOPHER_CALLER_ID)
}

export-modulemember -function Get-Robot
# SIG # Begin signature block
# MIIOWAYJKoZIhvcNAQcCoIIOSTCCDkUCAQExCzAJBgUrDgMCGgUAMGkGCisGAQQB
# gjcCAQSgWzBZMDQGCisGAQQBgjcCAR4wJgIDAQAABBAfzDtgWUsITrck0sYpfvNR
# AgEAAgEAAgEAAgEAAgEAMCEwCQYFKw4DAhoFAAQUZipUCC657JnxeIVMVE6Td184
# 99aggguPMIIFnDCCBISgAwIBAgIRAMRd9vOBG/0xAqGjaazZxhowDQYJKoZIhvcN
# AQELBQAwfDELMAkGA1UEBhMCVVMxCzAJBgNVBAgTAk1JMRIwEAYDVQQHEwlBbm4g
# QXJib3IxEjAQBgNVBAoTCUludGVybmV0MjERMA8GA1UECxMISW5Db21tb24xJTAj
# BgNVBAMTHEluQ29tbW9uIFJTQSBDb2RlIFNpZ25pbmcgQ0EwHhcNMTgwMjE2MDAw
# MDAwWhcNMjEwMjE1MjM1OTU5WjCB1TELMAkGA1UEBhMCVVMxDjAMBgNVBBEMBTIy
# OTA0MQswCQYDVQQIDAJWQTEYMBYGA1UEBwwPQ2hhcmxvdHRlc3ZpbGxlMRowGAYD
# VQQJDBFEeW5hbWljcyBCdWlsZGluZzEfMB0GA1UECgwWVW5pdmVyc2l0eSBvZiBW
# aXJnaW5pYTExMC8GA1UECwwoSW5mb3JtYXRpb24gVGVjaG5vbG9neSBhbmQgQ29t
# bXVuaWNhdGlvbjEfMB0GA1UEAwwWVW5pdmVyc2l0eSBvZiBWaXJnaW5pYTCCASIw
# DQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALKUqoPR5e6WUaa0SrPVpd1wuQsL
# nC+4w0bBzntuahVeISB23eRMhkkbzgziCbruhROdic5gHKmFOkh/TUMuh/muUaxX
# e3xsOyn6bNbxDOGVoeZQISfMzXEq6LVIIOmlTzYXHCtoy6aVx26zQRkEH/Yo82MG
# Pe5z+nLMXxXdCPcljPObC1qZBOqEnYTMugJf+kf+VcpTZDB0avb7uVPyePOOLH9F
# 3rmrd1FPe81cZqG/d+d1wlYXOYPWA6PFh854RL3ywowAsFQoe43U0wslK8/uuyik
# Sa8U3QIJSMToXFh62XkW7GUAPumjFSO9jMKqVlaFlWNvUAA6CiCZxk/MwJcCAwEA
# AaOCAb0wggG5MB8GA1UdIwQYMBaAFK41Ixf//wY9nFDgjCRlMx5wEIiiMB0GA1Ud
# DgQWBBSmIWLvwuUNZmlFeydZdTQYy4tNWzAOBgNVHQ8BAf8EBAMCB4AwDAYDVR0T
# AQH/BAIwADATBgNVHSUEDDAKBggrBgEFBQcDAzARBglghkgBhvhCAQEEBAMCBBAw
# ZgYDVR0gBF8wXTBbBgwrBgEEAa4jAQQDAgEwSzBJBggrBgEFBQcCARY9aHR0cHM6
# Ly93d3cuaW5jb21tb24ub3JnL2NlcnQvcmVwb3NpdG9yeS9jcHNfY29kZV9zaWdu
# aW5nLnBkZjBJBgNVHR8EQjBAMD6gPKA6hjhodHRwOi8vY3JsLmluY29tbW9uLXJz
# YS5vcmcvSW5Db21tb25SU0FDb2RlU2lnbmluZ0NBLmNybDB+BggrBgEFBQcBAQRy
# MHAwRAYIKwYBBQUHMAKGOGh0dHA6Ly9jcnQuaW5jb21tb24tcnNhLm9yZy9JbkNv
# bW1vblJTQUNvZGVTaWduaW5nQ0EuY3J0MCgGCCsGAQUFBzABhhxodHRwOi8vb2Nz
# cC5pbmNvbW1vbi1yc2Eub3JnMA0GCSqGSIb3DQEBCwUAA4IBAQBmmiGNgwIenWlf
# O6HPsrmfVVRZLAAbYWXqDgCVnxnRk0OvIwsXVIlby3J5Us9UCJHY4VfnTUsrygLG
# kptvcm4vCpmxpTv73/73ltxrW97KMdNKYNVbsKOK+cT/WJUGH6rQgXN6wEEH/5zi
# gkOBXxSCwQAer7tozjkWfiwFZuvKPdBz4dUeY1P8yS9cPNispFuql/U4zaAGw8Du
# aN7I2r0M/crKZOgqtTSTvFzcr6AAwcaly0HNmxixy0qpAr406VJ6Wyl16c2lIhk9
# s293qWEX92HMu4lOWwpf5LVeyemfgTBJ5GYDSMYm8H8Fj16XWnQrT6yvan1AR+Lh
# AEmQgU1XMIIF6zCCA9OgAwIBAgIQZeHi49XeUEWF8yYkgAXi1DANBgkqhkiG9w0B
# AQ0FADCBiDELMAkGA1UEBhMCVVMxEzARBgNVBAgTCk5ldyBKZXJzZXkxFDASBgNV
# BAcTC0plcnNleSBDaXR5MR4wHAYDVQQKExVUaGUgVVNFUlRSVVNUIE5ldHdvcmsx
# LjAsBgNVBAMTJVVTRVJUcnVzdCBSU0EgQ2VydGlmaWNhdGlvbiBBdXRob3JpdHkw
# HhcNMTQwOTE5MDAwMDAwWhcNMjQwOTE4MjM1OTU5WjB8MQswCQYDVQQGEwJVUzEL
# MAkGA1UECBMCTUkxEjAQBgNVBAcTCUFubiBBcmJvcjESMBAGA1UEChMJSW50ZXJu
# ZXQyMREwDwYDVQQLEwhJbkNvbW1vbjElMCMGA1UEAxMcSW5Db21tb24gUlNBIENv
# ZGUgU2lnbmluZyBDQTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAMCg
# L4seertqdaz4PtyjujkiyvOjduS/fTAn5rrTmDJWI1wGhpcNgOjtooE16wv2Xn6p
# Pmhz/Z3UZ3nOqupotxnbHHY6WYddXpnHobK4qYRzDMyrh0YcasfvOSW+p93aLDVw
# Nh0iLiA73eMcDj80n+V9/lWAWwZ8gleEVfM4+/IMNqm5XrLFgUcjfRKBoMABKD4D
# +TiXo60C8gJo/dUBq/XVUU1Q0xciRuVzGOA65Dd3UciefVKKT4DcJrnATMr8UfoQ
# CRF6VypzxOAhKmzCVL0cPoP4W6ks8frbeM/ZiZpto/8Npz9+TFYj1gm+4aUdiwfF
# v+PfWKrvpK+CywX4CgkCAwEAAaOCAVowggFWMB8GA1UdIwQYMBaAFFN5v1qqK0rP
# VIDh2JvAnfKyA2bLMB0GA1UdDgQWBBSuNSMX//8GPZxQ4IwkZTMecBCIojAOBgNV
# HQ8BAf8EBAMCAYYwEgYDVR0TAQH/BAgwBgEB/wIBADATBgNVHSUEDDAKBggrBgEF
# BQcDAzARBgNVHSAECjAIMAYGBFUdIAAwUAYDVR0fBEkwRzBFoEOgQYY/aHR0cDov
# L2NybC51c2VydHJ1c3QuY29tL1VTRVJUcnVzdFJTQUNlcnRpZmljYXRpb25BdXRo
# b3JpdHkuY3JsMHYGCCsGAQUFBwEBBGowaDA/BggrBgEFBQcwAoYzaHR0cDovL2Ny
# dC51c2VydHJ1c3QuY29tL1VTRVJUcnVzdFJTQUFkZFRydXN0Q0EuY3J0MCUGCCsG
# AQUFBzABhhlodHRwOi8vb2NzcC51c2VydHJ1c3QuY29tMA0GCSqGSIb3DQEBDQUA
# A4ICAQBGLLZ/ak4lZr2caqaq0J69D65ONfzwOCfBx50EyYI024bhE/fBlo0wRBPS
# Ne1591dck6YSV22reZfBJmTfyVzLwzaibZMjoduqMAJr6rjAhdaSokFsrgw5ZcUf
# TBAqesReMJx9THLOFnizq0D8vguZFhOYIP+yunPRtVTcC5Jf6aPTkT5Y8SinhYT4
# Pfk4tycxyMVuy3cpY333HForjRUedfwSRwGSKlA8Ny7K3WFs4IOMdOrYDLzhH9Jy
# E3paRU8albzLSYZzn2W6XV2UOaNU7KcX0xFTkALKdOR1DQl8oc55VS69CWjZDO3n
# YJOfc5nU20hnTKvGbbrulcq4rzpTEj1pmsuTI78E87jaK28Ab9Ay/u3MmQaezWGa
# Lvg6BndZRWTdI1OSLECoJt/tNKZ5yeu3K3RcH8//G6tzIU4ijlhG9OBU9zmVafo8
# 72goR1i0PIGwjkYApWmatR92qiOyXkZFhBBKek7+FgFbK/4uy6F1O9oDm/AgMzxa
# sCOBMXHa8adCODl2xAh5Q6lOLEyJ6sJTMKH5sXjuLveNfeqiKiUJfvEspJdOlZLa
# jLsfOCMN2UCx9PCfC2iflg1MnHODo2OtSOxRsQg5G0kH956V3kRZtCAZ/Bolvk0Q
# 5OidlyRS1hLVWZoW6BZQS6FJah1AirtEDoVP/gBDqp2PfI9s0TGCAjMwggIvAgEB
# MIGRMHwxCzAJBgNVBAYTAlVTMQswCQYDVQQIEwJNSTESMBAGA1UEBxMJQW5uIEFy
# Ym9yMRIwEAYDVQQKEwlJbnRlcm5ldDIxETAPBgNVBAsTCEluQ29tbW9uMSUwIwYD
# VQQDExxJbkNvbW1vbiBSU0EgQ29kZSBTaWduaW5nIENBAhEAxF3284Eb/TECoaNp
# rNnGGjAJBgUrDgMCGgUAoHgwGAYKKwYBBAGCNwIBDDEKMAigAoAAoQKAADAZBgkq
# hkiG9w0BCQMxDAYKKwYBBAGCNwIBBDAcBgorBgEEAYI3AgELMQ4wDAYKKwYBBAGC
# NwIBFTAjBgkqhkiG9w0BCQQxFgQUqIarfhjNWXTpyVQCuEtYPRM+1x8wDQYJKoZI
# hvcNAQEBBQAEggEAEzkedXaiI8UJpOdLZQJpmsaeeLXHrSULt/2GXK8u65V48TXu
# UMz4yfHFY+Bg3WOE+IObmoJvhteJuT5TqK2SNKdARtLdvyCxZebbMiOrbUrPLaSi
# pEsVZt4SKHMbbR3IDQlLAb5F/QrnLE6yq/27aCgxjcGkn3MGD7iMWueS1J6tIQf+
# VTAufhmwCVGCnCHaiZnakF+hpqnGJqjg3E+B0RODX/XqXOzPPJkMfLjXErSfG6iJ
# vL9khAG4Y5UOwhK9RCd7L6ihP+aMD1VzxVWLfpc8Kxb2qg0r5aYSzLTQtb44Olul
# XAQWpH/4wOxjKKWU5RNw9Fox8g/DvvM4kFVN5w==
# SIG # End signature block
