# Run and Deploy Your Robot

The same robot can move cleanly through these stages:

1. local workstation development
2. shared team development or review
3. long-running deployment on a server, VM, or container

The big v3 improvement is that you do not need a special container-only development environment to get there.

## Local runs

For day-to-day development, launch the engine from the robot home and keep editing `custom/`:

```bash
cd ~/robots/acme
/opt/gopherbot/gopherbot
```

## Long-running deployments

For long-lived robots, the most common approaches are:

- `systemd` on a VM or server
- a container in Kubernetes or another orchestrator

In those environments the robot typically boots from:

- a checked-out robot home already on disk, or
- `GOPHER_CUSTOM_REPOSITORY` plus `GOPHER_DEPLOY_KEY` so the engine can bootstrap itself

Use the next pages for the concrete deployment knobs.
