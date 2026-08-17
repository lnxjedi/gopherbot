# Container Deployment

Containers are a valid deployment target for Gopherbot.

## Good use cases

- you already operate Kubernetes or another container platform
- you want immutable engine images and environment-driven robot bootstrap
- your team already manages secrets and restart policy through an orchestrator

## What to keep in mind

- Build and test the robot locally first.
- Treat the container as a deployment wrapper around the same robot, not as a separate authoring environment.
- Provide the same critical environment values you would keep in `.env` on a VM.
- Avoid running multiple production instances of the same robot unless you have explicitly designed for that.

## Typical bootstrap model

An empty working directory inside the container is fine as long as you provide:

- `GOPHER_ENCRYPTION_KEY`
- `GOPHER_CUSTOM_REPOSITORY`
- `GOPHER_DEPLOY_KEY`
- any environment selector or connector overrides your robot depends on

On startup, Gopherbot will clone the robot repo, load config, and continue as a normal robot.
