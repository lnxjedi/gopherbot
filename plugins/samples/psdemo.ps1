#!powershell.exe
# NOTE: Gopherbot script plugins on Windows need to know what
# interpreter to use. If it's not in the path, use the full
# path to the interpreter, e.g.:
#!C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe

# For troubleshooting the escaping of args to PowerShell
#write-error "arg dump"
#$Args | write-error
#write-error "end dump"
# boilerplate
# Stylistic, can be omitted; $cmdArgs is always a String[],
# but $Args turns into a String when you shift off the 2nd item
[String[]]$cmdArgs = $Args
Import-Module "$Env:GOPHER_INSTALLDIR\lib\gopherbot_v1.psm1"
$bot = Get-Robot
# end boilerplate

$config = @'
Help:
- Keywords: [ "echo" ]
  Helptext: [ "(bot), echo <simple text> - trivially repeat a phrase" ]
- Keywords: [ "power" ]
  Helptext: [ "(bot), power (me!) - prove that PowerShell plugins work" ]
- Keywords: [ "listen" ]
  Helptext: [ "(bot), listen (to me!) - ask a question" ]
- Keywords: [ "remember", "memory" ]
  Helptext: [ "(bot), remember <anything> - prove the robot has a brain(tm)" ]
- Keywords: [ "recall", "memory" ]
  Helptext: [ "(bot), recall - prove the robot has a brain(tm)" ]
- Keywords: [ "forget", "memory" ]
  Helptext: [ "(bot), forget <#> - ask the robot to forget one of it's remembered 'facts'" ]
- Keywords: [ "check" ]
  Helptext: [ "(bot), check me - get the bot to check you out" ]
- Keywords: [ "network", "host", "ip" ]
  Helptext: [ "(bot), netinfo" ]
CommandMatchers:
- Command: "echo"
  Regex: '(?i:echo ([.;!\d\w-, ]+))'
- Regex: (?i:power( me)?!?)
  Command: power
- Regex: (?i:listen( to me)?!?)
  Command: listen
- Regex: (?i:remember ([-\w .,!?]+))
  Command: remember
- Regex: (?i:(?:recall|memories))
  Command: recall
- Regex: '(?i:forget #?([\d]{1,2}))'
  Command: forget
- Regex: (?i:check me)
  Command: check
- Regex: (?i:netinfo)
  Command: netinfo
- Regex: (?i:crash)
  Command: crash
Config:
  Replies:
  - "You've got THE POWAH!!"
  - "Ah, dang it - looks like the power is unplugged"
  - "By the power of Greyskull...  nah, I just can't do it"
  - "Sorry, you'll need to change my batteries first"
'@

# the equivalent of 'shift' for PowerShell
$command, $cmdArgs = $cmdArgs

switch ($command)
{
  "configure" {
    Write-Output $config
    exit
  }
  "remember" {
    [String] $thing = $cmdArgs[0]
    $bot.Say("Ok, I'll remember '$thing'")
    $memory = $bot.CheckoutDatum("memory", $TRUE)
    if ($memory.exists) {
      $memory.Datum += $thing
    } else {
      [String[]] $memory.Datum = @( $thing )
    }
    $ret = $bot.UpdateDatum($memory)
    if ($ret -ne "Ok") {
      $bot.Say("I'm having a hard time remembering things")
    }
  }
  "recall" {
    $memory = $bot.CheckoutDatum("memory", $FALSE)
    if ($memory.exists) {
      [String[]] $memories = @("Here's what I remember:")
      for ($i=0; $i -lt $memory.Datum.count; $i++){
        $memories += [String]($i + 1) + ": " + $memory.Datum[$i]
      }
      $bot.CheckinDatum($memory)
      $recollection = [String]::Join("`n", $memories)
      $bot.Say($recollection)
    } else {
      $bot.Say("Gosh, I don't remember ANYTHING!")
    }
  }
  "forget" {
    $i = [int]$cmdArgs[0] - 1
    $memory = $bot.CheckoutDatum("memory", $TRUE)
    [System.Collections.ArrayList]$memories = $memory.Datum
    if ($memories[$i] -ne $null) {
      $m = $memories[$i]
      $bot.Say("Ok, I'll forget $m")
      $memories.RemoveRange($i,$i)
      $memory.Datum = $memories
      $bot.UpdateDatum($memory)
    } else {
      $bot.Say("Gosh, I don't think I ever knew that!")
    }
  }
  "echo" {
    $bot.Log("Debug", "echo requested")
    # NOTE!!! In PowerShell, an array of strings with only one value is just a string
    $heard = $cmdArgs[0]
    $bot.Say("You said: $heard")
  }
  "listen" {
    $dbot = $bot.Direct()
    $dbj = $dbot | ConvertTo-Json
    $bot.Log("Debug", $dbj)
    $rep = $dbot.PromptForReply("SimpleString", "Ok, what do you want to tell me?")
    if ($rep.Ret -eq "Ok") {
      $dbot.Say("I heard you alright, you said: $rep")
    } else {
      $dbot.Say("I had a problem hearing you - funny characters?")
    }
  }
  "check" {
    if ( $bot.CheckAdmin() ) {
      $bot.Say("It looks like you're an admin, sweet!")
    } else {
      $bot.Say("Hrmph - well, you're not an administrator")
    }
    $bot.Pause(1)
    $bot.Say("Now I'll request elevation...")
    if ( $bot.Elevate($TRUE) ) {
      $bot.Say("Everything checks out, you're free to go")
    } else {
      $bot.Say("Huh - looks like you can't do any REAL work...")
    }
  }
  "power" {
    $firstName = $bot.GetSenderAttribute("firstName")
    $bot.Say("Sure, $firstName, hang on and I'll see what I can do")
    $bot.Pause(1.5)
    $cfg = $bot.GetTaskConfig()
    $bot.Say($bot.RandomString($cfg.Replies))
  }
  "netinfo" {
    $ni = ipconfig.exe
    $ni = [String]::Join("`n", $ni)
    $bot.Say("Here you go:")
    $bot.Say("$ni", "fixed")
  }
  # TODO: remove later, for troubleshooting Windows hangs
  "crash" {
    $bot.Say("Cool! Here we go...")
    write-error "crashing"
    exit 1
  }
}
# SIG # Begin signature block
# MIIOSwYJKoZIhvcNAQcCoIIOPDCCDjgCAQExCzAJBgUrDgMCGgUAMGkGCisGAQQB
# gjcCAQSgWzBZMDQGCisGAQQBgjcCAR4wJgIDAQAABBAfzDtgWUsITrck0sYpfvNR
# AgEAAgEAAgEAAgEAAgEAMCEwCQYFKw4DAhoFAAQUhSWwr6AOeMKM5q0JfLwmRyUX
# /+eggguCMIIFjzCCBHegAwIBAgIRAJJHZXGVpHVKJI9gIzdPrk8wDQYJKoZIhvcN
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
# DQEJBDEWBBRmXvsZGaGGPqD3NsVjbin10l75hTANBgkqhkiG9w0BAQEFAASCAQCb
# sGkACPGDOEEKUdoDhNg5FYNezHX3rGKeUA+2sU5HU5apf2WOkXQpDG3RAAw1GMm4
# uJbmJRpgoPeqIyauDwUid0k9o1TWpdS6lko8Y7Sdz7jdlAWhSlrITAkt/owzJMI+
# tf19lX7CE9mJpnwCWo4dG3VblPJPoEOYVPEUnc/U4o3w0WhEC2tW341+Bc+u2eUQ
# Yhi5besGyUwMQf4JIZy4fmW0cmkNxgPCEQZ94cVCrVho/3ERXK3KQrkrrgRJr9sL
# 4eY3tIoKjvCpOyEwlcJ2lbN9K03xuRJNIjG03iAxlNMtPHvesdpt3+sqd5wHPsFQ
# p7Pz8j/ko/csJnhHOTeG
# SIG # End signature block
