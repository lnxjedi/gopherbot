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
    - [Early Encryption Initialization](upgrade/Encryption.md)
    - [Long-Term Memories](upgrade/Memories.md)

- [Initial Robot Configuration](RobotInstall.md)
    - [Requirements](botsetup/Requirements.md)
    - [Manual Setup](botsetup/ManualSetup.md)
    - [Setup with Containers](botsetup/ContainerSetup.md)
    - [Using the Setup Plugin](botsetup/Plugin.md)
    - [Setting up a Robot with Gitpod](botsetup/Gitpod.md)

- [Deploying and Running Your Robot](RunRobot.md)
    - [CLI Operation](deploy/CLI.md)
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

## Gopherbot Development
- [Working on Gopherbot](GopherDev.md)
    - [Development Robot](botdev/DevelRobot.md) <!--TODO: write me! -->
    - [Integration Tests](botdev/IntegrationTests.md)
    - [Coding with Gitpod](botdev/Gitpod.md) <!--TODO: write me! -->
    - [Important Structs and Interfaces](botdev/StructsInterfaces.md)
