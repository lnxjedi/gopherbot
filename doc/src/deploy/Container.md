# Running in a Container

A major goal for **Gopherbot** v2 was container-native operation, and the ability to run your robot with all it's functionality and state/context, without requiring a persistent volume mount or a custom image with your robot baked-in. You can launch your robot using any of the [stock images](https://quay.io/lnxjedi/gopherbot); you should only need to create a custom image if your robot requires specific extra tools or libraries to do it's work, such as e.g. an ldap client or specific python module.

This section doesn't delve in to the specifics of running your robot in any particular container environment. Whether your robot runs as a persistent container on a Docker host, or as a managed container in a Kubernetes or Openshift cluster - or elsewhere - the same principles apply. If you're planning on deploying in a container, it is presumed that you're already running other containerized workloads, and can provide the requirements by common means for your container environment.

The contents of your robot's `.env` file are all that's needed - the built-in **bootstrap** plugin will use your robot's `deploy_key` key to clone it's `GOPHER_CUSTOM_REPOSITORY` configuration repository, then use the `GOPHER_ENCRYPTION_KEY` to decrypt the `manage_key` ssh private key to restore file-backed memories if you use the default `file` brain for the standard robot.

You can find example `docker` commands and comments regarding usage in `resources/docker/Makefile`.

## Container Considerations

### Backing up State and Memories
The standard robot includes a commented-out definition for scheduling the `backup` job in the stock `robot.yaml`. You should uncomment this and set a relatively frequent schedule for backups. The underlying job uses `git status --porcelain` to determine if a backup is needed, so there's little overhead in the case where memories don't change frequently. Since new memories tend to come in batches - for instance, adding bookmarks with the links plugin - an hourly schedule probably strikes a good balance between keeping the `robot-state` branch updated while not creating too many superfluous commits.

The **bootstrap** plugin will trigger a restore during start-up, so launching your robot into a new, empty container should restore it's state and memories automatically. Take care that you don't run multiple instances of your robot; not only would this run the risk of corrupting state, but this can produce strange behavior in your team chat.

### Providing Environment Variables
When launching your robot in a container, you'll need to provide the environment variables defined in the robot's `.env` file. There are two primary ways of accomplishing this:
* Both `docker` and `podman` allow you to set environment variables with a `--env-file .env` argument
* Container orchestration environments or e.g. `docker-compose` provide other means of providing these values; take care that the value for `GOPHER_ENCRYPTION_KEY` doesn't get committed to a repository

### Setting the `GOPHER_PROTOCOL` Environment Variable
When starting `gopherbot` at the CLI, or using systemd, the value for `GOPHER_PROTOCOL` should be commented out in the `.env`; this value is required for launching in a container so that your robot will start and connect to your team chat.
