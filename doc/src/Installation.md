# Installation

There are three distinct tasks involved in installing and running a **Gopherbot** robot:

1. This chapter discusses installing the **Gopherbot** distribution archive, normally in `/opt/gopherbot`; this provides the `gopherbot` binary, default configuration, and an assortment of included batteries (libraries, plugins, jobs, tasks, helper scripts and more); if you're using a [Gopherbot container](https://hub.docker.com/r/lnxjedi/gopherbot), installation is essentially a no-op.
1. Configuring a runnable instance of a robot for your team; the included **autosetup** plugin should make this an "easy button" - discussed in the chapter on [Initial Configuration](RobotInstall.md).
1. Deploying and running your robot on a server, VM, or in a container - covered in the chapter on [Running your Robot](RunRobot.md).

## **Gopherbot** and *Robots*

It's helpful to understand the relationship between **Gopherbot** and individual robots you run. It's helpful to compare Gopherbot with *Ansible*:
* *Gopherbot* is similar to *Ansible* - a common code base with an assortment of included batteries, but with limited functionality on it's own
* A *Robot* is comparable to a collection of playbooks and/or roles - this is your code for accomplishing work in your environment, which uses *Gopherbot* and it's included extensions to do it's work

Similar to Ansible playbooks and roles, individual robots may periodically require updates as you upgrade the **Gopherbot** core.