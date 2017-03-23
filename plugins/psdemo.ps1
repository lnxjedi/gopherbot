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
[String[]]$cmdArgs = $Args
Import-Module "$Env:GOPHER_INSTALLDIR\lib\Robot.psm1"
$channel, $user, $plugID, $cmdArgs = $cmdArgs
$bot = Get-Robot $channel $user $plugID
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
    # NOTE!!! In PowerShell, an array of strings with only one value is just a string
    $heard = $cmdArgs[0]
    $bot.Say("You said: $heard")
  }
  "multi"
  {
    Write-Error "Got:"
    Write-Error $cmdArgs[0]
    Write-Error $cmdArgs[1]
  }
}