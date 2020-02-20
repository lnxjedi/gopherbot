# Setting up a Robot with Gitpod

[Gitpod](https://gitpod.io) is an online development environment with a generous free offering for use with public free software projects. You can use **Gitpod** to configure a new robot, or to develop **Gopherbot** on your own fork. Once you've signed up for an account, you can use the **setup** plugin from the [Gitpod Online Demo](https://gitpod.io/#https://github.com/lnxjedi/gopherbot):

```
############################################################################
Welcome to the Gopherbot Demo. This will run Gopherbot
in terminal connector mode, where you can use the setup
plugin to configure a new robot and store it in a git
repository.
############################################################################

2020/02/20 15:06:57 Debug: Checking os.Stat for dir 'custom' from wd '': stat custom: no such file or directory
2020/02/20 15:06:57 Debug: Checking os.Stat for dir 'conf' from wd '': stat conf: no such file or directory
2020/02/20 15:06:57 Warning: Starting unconfigured; no robot.yaml/gopherbot.yaml found
Terminal connector running; Use '|c<channel|?>' to change channel, or '|u<user|?>' to change user
c:general/u:alice -> 2020/02/20 15:06:58 Warning: GOPHER_CUSTOM_REPOSITORY not set, not bootstrapping
general: *******
general: Welcome to the *Gopherbot* terminal connector. Since no configuration
was detected, you're connected to 'floyd', the default robot.
general: If you've started the robot by mistake, just hit ctrl-D to exit and try
'gopherbot --help'; otherwise feel free to play around with the default robot -
you can start by typing 'help'. If you'd like to start configuring a new robot,
type: ';setup'.
c:general/u:alice -> ;setup
...
```
