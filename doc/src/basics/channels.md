# Availability by Channel
Individual **Gopherbot** robots normally limit commands to certain channels, contextually; for instance, the "build" command may be limited to a job channel, where developers can all "see" each other starting builds. Generally speaking, limiting commands to select channels is fairly common for a **Gopherbot** robot. By adding `Channels: [ "foo", "bar" ]` to a plugin's `*.yaml` configuration file, an administrator can easily override the default channels for a given plugin.

## Default Channels
Many of the robot's plugins aren't context-sensitive, and can be available anywhere. Practically, however, you might want to limit these to a small number of channels to prevent common channels from getting junked up with robot noise. The slack protocol configuration for the standard robot has:
```
DefaultChannels: [ "general", "random" ]
```