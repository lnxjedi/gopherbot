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
- Regex: (?i:forget ([\d]{1,2}))
  Command: forget
- Regex: (?i:check me)
  Command: check
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
      for ($i=0; $i -lt $memory.Datum.length(); $i++){
        $memories += [String]($i + 1) + ": " + $memories[$i]
      }
      $bot.CheckinDatum($memory)
      $recollection = [String]::Join("\n", $memories)
      $bot.Say($memories)
    } else {
      $bot.Say("Gosh, I don't remember ANYTHING!")
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
  "power" {
    $firstName = $bot.GetSenderAttribute("firstName")
    $bot.Say("Sure, $firstName - You've got THE POWAH!!")
  }
}