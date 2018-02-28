# Getting Gopherbot

## Installation Notes

### Install and Config Directories
Gopherbot combines configuration data from two directories; the `install` directory (normally `C:\Program Files\Gopherbot`), and the `config` directory (normally `C:\ProgramData\Gopherbot`). Configuration in the `install` directory should be specific to a given robot / environment (e.g. *dev* and *prod*), and normally contains e.g. credentials. The contents of the `config` directory would normally be kept in a git repository, and would contain common configuration as well as locally developed script plugins. See [Configuration](Configuration.md).

### Windows Security
Depending on the security settings of your Windows host, you may need to unblock or otherwise allow execution of `gopherbot.exe` and/or the PowerShell library and plugins. A full discussion of Windows security features and correct configuration is outside the scope of this document. Note, however, that the PowerShell library and sample plugins have been signed with a code-signing certificate.

## Binary releases
Pre-compiled archives for Windows are available for download directly from github:
https://github.com/uva-its/gopherbot/releases

## Installing from source
If you have a Go environment set up on your machine, you can can `go get github.com/uva-its/gopherbot` and create a `.zip` archive. See the `mkdist.sh` script for a listing of what should be in the archive.

# Production Installation of Gopherbot as a Windows Service

1. Create a `C:\Program Files\Gopherbot` install directory, and unzip the **gopherbot-\*.zip** archive there
1. Create a `C:\ProgramData\Gopherbot` config directory, copy the `conf` and `brain` directories there from the install directory
1. Rename `C:\Program Files\Gopherbot\conf\gopherbot.yaml.sample` to `gopherbot.yaml`, and similar with `C:\ProgramData\Gopherbot\conf\gopherbot.yaml.sample`
1. If you haven't already, get a Slack token for your robot from `https://\<your-workspace\>.slack.com/services/new/bot`
1. Open an editor as local administrator and edit `C:\Program Files\Gopherbot\conf\gopherbot.yaml`, uncommenting and modifying `Protocol` and `ProtocolConfig`
1. Edit `C:\ProgramData\Gopherbot\conf\gopherbot.yaml` and uncomment and/or modify:
  * `AdminContact`
  * `DefaultChannels`
  * `AdminUsers`
  * `Alias`

**NOTE: Individual administrators may choose to configure different items in different locations**

7. Optionally uncomment `ExternalPlugins` and the stanza for `psdemo.ps1` to enable the sample PowerShell plugin
1. Open a command prompt or powershell window as administrator
1. Change directories to the install directory: `cd C:\Program Files\Gopherbot`
1. Install Gopherbot as a service: `.\gopherbot.exe -w install`
1. Start the Gopherbot service: `.\gopherbot.exe -w start`

At this point you can use `services.msc` to start & stop Gopherbot, or to configure it for Automatic start.

# Upgrading

Upgrading to the lastest version should be simple:
1. Stop the `Gopherbot` service
1. Unzip a new archive in `C:\Program Files\Gopherbot`
2. Unblock or allow execution as necessary
3. Re-start the `Gopherbot` service
