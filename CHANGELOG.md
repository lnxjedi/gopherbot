# v2.1.5
* Stronger TOTP implementation; generate user codes with the CLI and add encrypted secrets to `conf/plugins/builtin-totp.yaml`
* Make standard "robot, help" contextual (less noisy); "robot, help-all" gives the formerly verbose output
* Export `PYTHONPATH` and `RUBYLIB`, removed ugly env-var references from python & ruby plugins
* Simple tasks now inherit their memory namespace from the pipeline they're added to

# v2.1.4
* Allow for ephemeral memories with the git brain; any memories with leading underscores will be ignored by the '.gitignore', allowing developers to prevent frequent git commits for fast changing memories. Also made the .gitignore algorithm more robust.

# v2.1.3
* More container build updates - make all containers more useful for dev env

# v2.1.2
* Update container builds, using GitHub actions for releases and containers

# v2.1.1
* Compile in the dynamodb brain, famously used by Floyd
* Update aws and other deps

# v2.1.0
Changes since v2.0.2:

The major update for 2.1.0+ is the temporary and possibly permanent removal of modular builds. Support for loadable Go modules was *cool*, but it meant that I couldn't build a single distribution archive installable on all recent Linux distributions due to modular builds using the gcc toolchain linker. Starting with 2.1.0, **Gopherbot** will once again create installable build artifacts.

The other major change is a greater focus on **Slack** as the primary protocol. While the **terminal** connector will continue to be the best way to develop Gopherbot jobs and plugins, the primary build artifacts will be largely Slack-specific.

Other updates/fixes:
* Several fixes to the `autosetup` installer for additional robustness.
