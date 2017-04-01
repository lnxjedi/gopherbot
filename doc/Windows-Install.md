# Getting Gopherbot

## Binary releases
Pre-compiled archives for Windows are available for download directly from github:
https://github.com/uva-its/gopherbot/releases

## Installing from source
If you have a Go environment set up on your machine, you can can `go get github.com/uva-its/gopherbot`. To create an install archive run `.\mkdist.ps1` in `%GOPATH%\src\github.com\uva-its\gopherbot`.

# Production Installation of Gopherbot as a Windows Service

1. Create a `C:\Program Files\Gopherbot` install directory, and unzip the gopherbot-*.zip archive there
2. Create a `C:\Windows\gopherbot` config directory and copy the `conf` and `brain` directories there from the install directory
3. If you haven't already, get a Slack token for your robot from https://\<your-team\>.slack.com/services/new/bot
5. Open an editor as local administrator and edit `C:\Windows\gopherbot\conf\gopherbot.yaml`, uncommenting and/or modifying:
  * `AdminContact`
  * `DefaultChannels`
  * `AdminUsers`
  * `Alias`
  * `Protocol` and `ProtocolConfig`
6. Optionally uncomment `ExternalPlugins` and the stanza for `psdemo.ps1` to enable the sample PowerShell plugin
7. Open a command prompt or powershell window as administrator
8. Change directories to the install directory: `cd C:\Program Files\Gopherbot`
9. Install Gopherbot as a service: `.\gopherbot.exe -w install`
10. Start the Gopherbot service: `.\gopherbot.exe -w start`

At this point you can use `services.msc` to start & stop Gopherbot, or to configure it for Automatic start
