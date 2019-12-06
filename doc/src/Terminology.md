# Terminology

This section is most important for referring back to as you read the documentation, to disambiguate terms.

(It's also for me, to help maintain some consistency)

* **Gopherbot** - The installed software archive that comprises the **Gopherbot DevOps Chatbot**
* **robot** - A configured instance of **Gopherbot** available in your team chat; possibly also a reference to a particular robot's configuration repository
* **plugin** (or **command plugin**) - A piece of code that provides new commands or code for authorization and/or elevation
* **authorizer** - special plugin command used to determine whether a given user is authorized for a given command, normally checking some kind of group membership
* **elevator** - special plugin command providing additional verification of user identity; this can be as simple as a totp token or [Duo](https://duo.com) two-factor, or as complex as prompting another user before allowing a command to proceed
* **job** - jobs are pieces of code that typically use the pipeline API for creating pipelines to perform complex scheduled tasks such as backups and monitoring, or for software builds that may be triggered by hosted git service integrations with your chat platform
* **task** - tasks are small pieces of code that generally form the parts of a pipeline, such as initializing (and tearing down) the `ssh-agent`, running pipeline scripts, or sending notifications; the task is also the base object for jobs and plugins, so "task" may refer to any entry in `ExternalPlugins`, `ExternalTasks`, `ExternalJobs`, etc. 
* **Robot** - The object passed to user plugins, jobs and tasks; also the **Go** library for loadable modules
