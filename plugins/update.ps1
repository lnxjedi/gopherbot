#!powershell.exe
# NOTE: Gopherbot script plugins on Windows need to know what
# interpreter to use. If it's not in the path, use the full
# path to the interpreter, e.g.:
#!C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe

# update.ps1 - a PowerShell plugin allowing the robot to
# update it's Configuration Directory using Git for Windows.
# It's up to the bot admin to install an ssh keypair for the
# bot in the %USERPROFILE%\.ssh directory that has at least
# read access to the git repository. (normally a deploy key)

[String[]]$cmdArgs = $Args
Import-Module "$Env:GOPHER_INSTALLDIR\lib\gopherbot_v1.psm1"
$bot = Get-Robot

# default plugin configuration, yaml data
$config = @'
RequireAdmin: true
Channels: [ 'botadmin' ]
ElevatedCommands: [ 'update' ]
AllowDirect: true
Help:
- Keywords: [ "config", "configuration", "update" ]
  Helptext: [ "(bot), update configuration - perform a 'git pull' in the configuration directory" ]
CommandMatchers:
- Command: "update"
  Regex: '(?i:update config(?:uration)?)'
'@

# the equivalent of 'shift' for PowerShell
$command, $cmdArgs = $cmdArgs

switch ($command)
{
  "configure" {
    Write-Output $config
    exit
  }
  "update" {
    $bot.Say("Ok, I'll issue a git pull...")
    $pdrive, $ppath = $env:GOPHER_CONFIGDIR.split(":")
    $env:HOMEDRIVE = "${pdrive}:"
    $env:HOMEPATH = $ppath
    Set-Location $env:GOPHER_CONFIGDIR
    $gitpath = $env:ProgramFiles -replace ' ','` '
    $result = Invoke-Expression "$gitpath\Git\bin\git.exe pull" 2>&1 | Out-String
    $bot.Say("Operation completed with result:")
    $bot.Say("$result", "fixed")
  }
}

# SIG # Begin signature block
# MIIOWAYJKoZIhvcNAQcCoIIOSTCCDkUCAQExCzAJBgUrDgMCGgUAMGkGCisGAQQB
# gjcCAQSgWzBZMDQGCisGAQQBgjcCAR4wJgIDAQAABBAfzDtgWUsITrck0sYpfvNR
# AgEAAgEAAgEAAgEAAgEAMCEwCQYFKw4DAhoFAAQU3pGFWnkiw/mr1ggdqK3rmFDW
# C4+ggguPMIIFnDCCBISgAwIBAgIRAMRd9vOBG/0xAqGjaazZxhowDQYJKoZIhvcN
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
# NwIBFTAjBgkqhkiG9w0BCQQxFgQUq4yXDo+u5/YWg+fS+Whm8l1F5zEwDQYJKoZI
# hvcNAQEBBQAEggEAKjiZDfWJ4ez3TtK6QVwc37zZdNEzxcG/sFTKCAZIICNPHgoO
# CFqThsOZUZNXSk67/SePBjrwbIK8/dRrgkGtr6pvRFRJFIQPHsRqMyOBc7Zvc2i3
# ZvNP2rl18Zx3SKnsIBk/ZSzGHpi/YgxztROFd4/Is2/6nHnegHaqBATk8cPKHGMD
# oAssXikYbFc3eD33IJt7dp9Q6WAUA9tA7xD4kYJWFNCOz3k7Hp/KSyFRDWckDY+U
# QN59uakVvVHwomtIr6lbCuE4IchpiZsh7DaugTjhx8YvuUcFmz4ehmGfnLNN+Saw
# Y2LrdxsStCxYzeDz5eOhF74tKmWKrIajhiUEkw==
# SIG # End signature block
