# Terminology

This section is the vocabulary for the rest of the manual.

- **Gopherbot**: the engine installation or source tree that contains the binary, built-in defaults, libraries, stock plugins, and robot skeleton.
- **robot**: a configured running instance of Gopherbot for one team or one purpose.
- **Robot**: the API object passed to plugin, job, and task handlers.
- **install tree**: the directory that contains the `gopherbot` executable plus shipped `conf/`, `lib/`, `resources/`, `plugins/`, `jobs/`, `tasks/`, and `robot.skel/`.
- **robot home**: the working directory where you launch the engine for one robot. In practice this is the directory that may contain `.env`, `.setup-state`, `workspace/`, and `custom/`.
- **custom config**: the robot-specific overlay under `custom/`. Installed defaults load first, then `custom/conf/...` overrides them.
- **default robot**: the built-in demo robot you get when no custom repository or custom config has been loaded yet. By default this robot is named `floyd`.
- **bootstrap**: the startup path where Gopherbot has enough environment to know which robot repository it should clone, but does not have the robot's config on disk yet.
- **plugin**: interactive automation triggered by chat messages.
- **job**: scheduled or triggered automation that usually assembles a pipeline of tasks.
- **task**: a reusable unit of work inside a pipeline. Some tasks are stock and included with the engine; others are custom.
- **pipeline**: the ordered execution chain built by a plugin or job.
- **primary protocol**: the connector that provides the robot's main runtime identity and default message context.
- **secondary protocol**: any additional connector enabled alongside the primary one.
- **UserRoster**: the global user directory for the robot. Identity and authorization decisions are username-based.
- **ProtocolConfig**: the connector-local configuration block loaded from `conf/protocols/<protocol>.yaml`.
- **BasicMarkdown**: the default cross-protocol outgoing message format in v3.
- **parameter**: a name/value setting attached to a task, plugin, job, or namespace. Secure parameters may need to be retrieved through the Robot API instead of plain environment variables.
- **namespace**: a reusable group of parameters shared across related automation.
- **environment file**: `.env` or `private/environment`, loaded early at startup. This is where deployment-specific settings such as `GOPHER_ENVIRONMENT`, `GOPHER_CUSTOM_REPOSITORY`, or secrets usually live.

You will also see these names throughout the docs:

- **Floyd**: the name of the shipped default robot.
- **Clu**: the long-running development robot used while building v3.
- **robot.skel**: the standard starting point for a new robot repository.
