# Gopherbot DevOps Chatbot

[Title](Title.md)

[Foreward](Foreward.md)

[Introduction](Introduction.md)

[Terminology](Terminology.md)

- [Gopherbot Software Installation](Installation.md)
    - [Requirements](install/Requirements.md)
    - [Manual Installation](install/ManualInstall.md)

- [Upgrading from Version 1](Upgrading.md)
    - [Required Bot Info](upgrade/BotInfo.md)
    - [External Plugin Configuration](upgrade/External-Plugin.md)
    - [Custom Configuration Directory](upgrade/Custom-Dir.md)
    - [Main Configuration File Rename](upgrade/robot-yaml.md)
    - [Early Encryption Initialization](upgrade/Encryption.md)
    - [Long-Term Memories](upgrade/Memories.md)

- [Initial Robot Configuration](RobotInstall.md)
    - [Requirements](botsetup/Requirements.md)
    - [Robot Directory Structure](botsetup/gopherhome.md)
    - [Quick Start with Autosetup](botsetup/Plugin.md)
    - [Setting up a Robot with Gitpod](botsetup/Gitpod.md)
    - [Manual Setup](botsetup/ManualSetup.md)
        - [Create the `GOPHER_HOME` directory](botsetup/bothome.md)
        - [Create the Initial `.env` File](botsetup/initenv.md)
        - [Initialize Encryption](botsetup/initcrypt.md)
        - [Copy the Standard Robot](botsetup/copystd.md)
        - [Generate SSH Keypairs](botsetup/sshkeys.md)
        - [Finish the `.env` File](botsetup/finalenv.md)
        - [Connect Robot to Team](botsetup/connect.md)
        - [Saving Your Robot to Git](botsetup/saverobot.md)
        - [Finished](botsetup/finished.md)
    - [Setup with Containers](botsetup/ContainerSetup.md)

- [Deploying and Running Your Robot](RunRobot.md)
    - [CLI Operation](deploy/CLI.md)
        - [Local Install](deploy/local.md)
        - [Container Operation](deploy/containercli.md)
        - [Using Gitpod](deploy/gitpodcli.md)
        - [Encrypting Secrets](deploy/secrets.md)
    - [Running with Systemd](deploy/systemd.md)
    - [Running in a Container](deploy/Container.md)

- [Configuring Gopherbot](Configuration.md)
    - [Environment Variables](Environment-Variables.md)
    - [Configuration File Loading](config/file.md)
    - [Job and Plugin Configuration](config/job-plug.md)
    - [Troubleshooting](config/troubleshooting.md)

- [Administering Your Robot](Admin.md)
    - [Administrator Commands](usage/admin.md)
    - [Command-Line Use](usage/cli.md)
    - [Logging](usage/logging.md)

- [Developing Extensions for Your Robot](botprogramming.md)

- [Gopherbot API](api/API-Introduction.md)
    - [Language Templates](api/Languages.md)
    - [Attribute Retrieval](api/Attribute-Retrieval-API.md)
    - [Brain Methods](api/Brain-API.md)
    - [Message Sending](api/Message-Sending-API.md)
    - [Pipeline Construction](api/Pipeline-API.md)
    - [Requesting Responses](api/Response-Request-API.md)
    - [Utility](api/Utility-API.md)

- [Module Support](Modules.md)

- [Jobs and Pipelines](pipelines/jobspipes.md)
    - [Included Tasks](pipelines/tasks.md)
    - [Task Environment Variables](pipelines/TaskEnvironment.md)
    - [Tool Integrations](pipelines/integrations.md)
    - [Integrating with SSH](pipelines/ssh.md)

## Appendix
- [Appendix](appendices/Appendix.md)
    - [A - Protocols](appendices/Protocols.md)
        - [A.1 - Slack](appendices/slack.md)
        - [A.2 - Rocket.Chat](appendices/rocket.md)
        - [A.3 - Terminal](appendices/terminal.md)
        - [A.4 - Test](appendices/testproto.md)
        - [A.5 - Nullconn](appendices/nullconn.md)

## Gopherbot Development
- [Working on Gopherbot](GopherDev.md)
    - [Development Robot](botdev/DevelRobot.md) <!--TODO: write me! -->
    - [Integration Tests](botdev/IntegrationTests.md)
    - [Coding with Gitpod](botdev/Gitpod.md) <!--TODO: write me! -->
    - [Important Structs and Interfaces](botdev/StructsInterfaces.md)
    - [Protocols](botdev/protocols.md)
