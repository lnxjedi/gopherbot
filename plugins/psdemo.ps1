#!powershell.exe
#!C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe
# NOTE: Gopherbot script plugins on Windows need the full
# path to the interpreter

# boilerplate
Import-Module "$Env:GOPHER_INSTALLDIR\lib\Robot.psm1"
$channel, $user, $plugID, $Args = $Args
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

$command, $Args = $Args

switch ($command)
{
  "configure"
  {
    Write-Output $config
    exit
  }
  "echo"
  {
    Write-Error "Got: $Args[0]"    
  }
  "multi"
  {
    Write-Error "Got:"
    Write-Error $Args[0]
    Write-Error $Args[1]
  }
}