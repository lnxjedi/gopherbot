# Setup with Containers

If you have an environment where you can run and interact with containers in a terminal window, using either [Docker](https://www.docker.com/) or [Podman](https://podman.io/), you can use the **setup** plugin with the default robot in a container:

**With Podman**:
```
$ podman run -it lnxjedi/gopherbot:latest
Trying to pull docker.io/lnxjedi/gopherbot:latest...
Getting image source signatures
...
Info: PID == 1, spawning child
Debug: Successfully raised privilege permanently for 'exec child process' thread 1; new r/euid: 1/1
Info: Starting pid 1 signal handler
Debug: Checking os.Stat for dir 'custom' from wd '': stat custom: no such file or directory
Debug: Checking os.Stat for dir 'conf' from wd '': stat conf: no such file or directory
Warning: Starting unconfigured; no robot.yaml/gopherbot.yaml found
Terminal connector running; Use '|c<channel|?>' to change channel, or '|u<user|?>' to change user
c:general/u:alice -> Warning: GOPHER_CUSTOM_REPOSITORY not set, not bootstrapping
general: *******
general: Welcome to the *Gopherbot* terminal connector. Since no configuration was detected, you're
connected to 'floyd', the default robot.
general: If you've started the robot by mistake, just hit ctrl-D to exit and try 'gopherbot --help';
otherwise feel free to play around with the default robot - you can start by typing 'help'. If you'd
like to start configuring a new robot, type: ';setup'.
c:general/u:alice -> ;setup
...
```

**With Docker**:
```
$ docker run -it lnxjedi/gopherbot:latest
Unable to find image 'lnxjedi/gopherbot:latest' locally
latest: Pulling from lnxjedi/gopherbot
...
Info: PID == 1, spawning child
Debug: Successfully raised privilege permanently for 'exec child process' thread 1; new r/euid: 1/1
Info: Starting pid 1 signal handler
Debug: Checking os.Stat for dir 'custom' from wd '': stat custom: no such file or directory
Debug: Checking os.Stat for dir 'conf' from wd '': stat conf: no such file or directory
Warning: Starting unconfigured; no robot.yaml/gopherbot.yaml found
Terminal connector running; Use '|c<channel|?>' to change channel, or '|u<user|?>' to change user
c:general/u:alice -> Warning: GOPHER_CUSTOM_REPOSITORY not set, not bootstrapping
general: *******
general: Welcome to the *Gopherbot* terminal connector. Since no configuration was detected, you're
connected to 'floyd', the default robot.
general: If you've started the robot by mistake, just hit ctrl-D to exit and try 'gopherbot --help';
otherwise feel free to play around with the default robot - you can start by typing 'help'. If you'd
like to start configuring a new robot, type: ';setup'.
c:general/u:alice -> ;setup
...
```
