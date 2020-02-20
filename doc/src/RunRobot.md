# Deploying and Running Your Robot

**Gopherbot** is very flexible about being able to bootstrap and run in a variety of environments. This chapter discusses the three primary ways you'll run your robot:
* It's helpful if you have access to a Linux workstation, VM, or container where you can bootstrap your robot and use [CLI commands](deploy/CLI.md)
   * Microsoft's free [VisualStudio Code](https://code.visualstudio.com/) has a remote extension that will allow you to edit files directly in a container, and the robot's adminstrator **save** command can push updates to the robot's git repository
   * If you robot is stored in a public [Github](https://github.com) repository, you can use [Gitpod](https://gitpod.io) to work with your robot at the CLI
* The `resources/` directory contains a template `robot.service` that can be used to run your robot on a Linux host [using Systemd](deploy/systemd.md)
* Using [Podman](https://podman.io), [Docker](https://docker.com), [Kubernetes](https://kubernetes.io), or any number of other container-centric environments, you can bootstrap and run your robot [in a Container](deploy/Container.md)
