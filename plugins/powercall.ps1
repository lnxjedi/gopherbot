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
- Keywords: [ "power" ]
  Helptext: [ "(bot), call power - call the power command from the psdemo plugin" ]
CommandMatchers:
- Command: "power"
  Regex: '(?i:call power)'
'@

# the equivalent of 'shift' for PowerShell
$command, $cmdArgs = $cmdArgs

switch ($command)
{
  "configure" {
    Write-Output $config
    exit
  }
  "power" {
    $bot.Say("Ok, I'll give the psdemo plugin a kick...")
    $status = $bot.CallPlugin("psdemo", @("power"))
    if ( $status -ne "Normal" ) {
      $bot.Reply("Hrm, I don't think psdemo did it's job!")
    }
  }
}
