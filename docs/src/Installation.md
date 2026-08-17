# Build and Install

The v3 installation story is simple:

1. Install the engine once on a Linux machine you control.
2. Create one or more robot working directories.
3. Run the engine from those directories while you develop locally.
4. When the robot is ready, deploy the same robot configuration to a VM, server, or container.

## Two supported ways to install the engine

- Build from source and keep the checkout intact.
- Unpack a release archive into a stable install location such as `/opt/gopherbot`.

In both cases, the `gopherbot` binary expects the rest of the install tree to be nearby. Treat the installation as a directory tree, not as a single standalone executable.

## Recommended layout

- Engine install tree: `/opt/gopherbot` or a source checkout you build in place
- Robot home: one directory per robot, for example `/srv/robots/acme` or `~/robots/acme`

You run the binary from the install tree while your current working directory is the robot home. That gives the engine access to both worlds:

- installed defaults from the engine tree
- writable custom config from the robot home

Continue with [Requirements](install/Requirements.md) if you are starting fresh, or skip ahead to [Create Your First Robot](RobotSetup.md) if you already have a working install.
