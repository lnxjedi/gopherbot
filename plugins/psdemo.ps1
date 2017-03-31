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
Import-Module "$Env:GOPHER_INSTALLDIR\lib\Robot.psm1"
$bot = Get-Robot
# end boilerplate

$config = @'
Channels: [ "random" ]
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
    $mjson = convertto-json $memory
    $bot.Log("Debug", "mjson before: $mjson")
    if ($memory.exists) {
      $memory.Datum += $thing
    } else {
      [String[]] $memory.Datum = @( $thing )
    }
    $mjson = convertto-json $memory
    $bot.Log("Debug", "mjson after: $mjson")
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
    $dbot.Say("Ok, what do you want to tell me?")
    $rep = $dbot.WaitForReply("SimpleString")
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
    $cfg = $bot.GetPluginConfig()
    $bot.Say($bot.RandomString($cfg.Replies))
  }
  "netinfo" {
    $ni = ipconfig.exe
    $ni = [String]::Join("`n", $ni)
    $bot.Say("Here you go:`n$ni")
  }
  # TODO: remove later, for troubleshooting Windows hangs
  "crash" {
    $bot.Say("Cool! Here we go...")
    write-error "crashing"
    exit 1
  }
}