# Robot Basics

Central to the design of Gopherbot is the idea that once your robot connects to your team chat and joins one or more channels, the robot "hears" every message in those channels, just as a user would. Every time the robot hears a message[^1]:
* It checks to see if the message was directed to it by name
* It checks for "ambient" message matches
* It checks for matching job triggers

This chapter focuses on the first case - sending commands directly to your robot.

[^1]: Depending on the configured values for `IgnoreUnlistedUsers` and `IgnoreUsers`, messages may be dropped entirely without any processing. This would show up in the logs at log level **Debug**.