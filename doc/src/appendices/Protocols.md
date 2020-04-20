# Appendix A: Protocols

**Gopherbot** communicates with users via different protocols, and extensions can modify their behavior and provide protocol-specific functionality based on values provided to the extension. For **Go** extensions, this is provided in the `robot.Message` struct provided by the `GetMessage()` method. For external scripts, this is provided in the `GOPHER_PROTOCOL` environment variable.

The design of **Gopherbot** is meant for mainly protocol-agnostic functionality. Running jobs and querying infrastructure should operate in much the same way whether the protocol is **Slack** or **Terminal**. However, some teams may wish to create protocol-specific extensions, and accept the risk of a more difficult transition should the team switch their primary chat platform.