# Gopherbot v2.1.0-pre

Changes since v2.0.2:

The major update for 2.1.0+ is the temporary and possibly permanent removal of modular builds. Support for loadable Go modules was *cool*, but it meant that I couldn't build a single distribution archive installable on all recent Linux distributions due to modular builds using the gcc toolchain linker. Starting with 2.1.0, **Gopherbot** will once again create installable build artifacts.

The other major change is a greater focus on **Slack** as the primary protocol. While the **terminal** connector will continue to be the best way to develop Gopherbot jobs and plugins, the primary build artifacts will be largely Slack-specific.

Other updates/fixes:
* Don't comment-out `GOPHER_PROTOCOL` during setup; fixes an issue with setting up new robots.