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
CommandMatchers:
- Command: "echo"
  Regex: '(?i:echo ([.;!\d\w-, ]+))'
- Command: "multi"
  Regex: '(?i:multi ([.;!\d\w-,]+) ([.;!\d\w-, ]+))'
'@

# the equivalent of 'shift' for PowerShell
$command, $cmdArgs = $cmdArgs

switch ($command)
{
  "configure"
  {
    Write-Output $config
    exit
  }
  "echo"
  {
    $bot.Log("Debug", "echo requested")
    # NOTE!!! In PowerShell, an array of strings with only one value is just a string
    $heard = $cmdArgs[0]
    $bot.Say("You said: $heard")
  }
  "multi"
  {
    $h1, $h2 = $cmdArgs
    $bot.Say("You said $h1 and $h2")
  }
}