# Setup with Containers

If you have an environment where you can run and interact with containers in a terminal window, using either [Docker](https://www.docker.com/) or [Podman](https://podman.io/), you can supply an answerfile for using the **autosetup** plugin in a container.

## Creating `answerfile.txt`

For container-based installs, you can start an empty container and use the `setup` command to display an appropriate `answerfile.txt`, using **Podman** to set up **slack** as an example:
```
$ podman run -it --name ansfile quay.io/lnxjedi/gopherbot:latest
Info: PID == 1, spawning child
Info: Starting pid 1 signal handler
Logging to robot.log; warnings and errors duplicated to stdout
Terminal connector running; Use '|c<channel|?>' to change channel, or '|u<user|?>' to change user
general: *******
general: Welcome to the *Gopherbot* terminal connector. Since no configuration was detected, you're
connected to 'floyd', the default robot.
general: If you've started the robot by mistake, just hit ctrl-D to exit and try 'gopherbot --help';
otherwise feel free to play around with the default robot - you can start by typing 'help'. If you'd like
to start configuring a new robot, type: ';setup <protocol>'.
c:general/u:alice -> ;setup slack
general: Copy to answerfile.txt:
<-- snip answerfile.txt -->
## Note on commenting conventions:
...
<-- /snip -->

Edit your 'answerfile.txt' and run the container with '--env-file answerfile.txt'.
Exiting (press <enter> ...)
c:general/u:alice -> 
Terminal connector finished
# Remove the temporary container
$ podman container rm ansfile
```

Once you have your `answerfile.txt` you'll need to edit the file and set appropriate values for each `ANS_*` variable appropriate to your robot. Take care to use an unquoted value for `ANS_ROBOT_ALIAS`.

## Starting the *autosetup* plugin

With a complete answerfile, you can start a *Gopherbot* container, passing the answerfile in for the environment. After the *autosetup* plugin runs and generates configuration, it'll pause and provide instructions for completing setup.

**With Podman**:
```
$ podman run -it --env-file answerfile.txt quay.io/lnxjedi/gopherbot:latest
Trying to pull quay.io/lnxjedi/gopherbot:latest...
Getting image source signatures
...
Info: PID == 1, spawning child
Info: Starting pid 1 signal handler
Info: Logging to robot.log
null connector: Initializing encryption and restarting...
Info: Logging to robot.log
null connector: Continuing automatic setup...
...
```

**With Docker**:
```
$ docker run -it --env-file answerfile.txt quay.io/lnxjedi/gopherbot:latest
Unable to find image 'quay.io/lnxjedi/gopherbot:latest' locally
latest: Pulling from quay.io/lnxjedi/gopherbot
...
Info: PID == 1, spawning child
Info: Starting pid 1 signal handler
Info: Logging to robot.log
null connector: Initializing encryption and restarting...
Info: Logging to robot.log
null connector: Continuing automatic setup...
...
```

... then follow the provided instructions to set yourself up as an administrator and save your robot's configuration to a *git* repository.

## Finishing Up
After your robot has successfully saved it's configuration to it's *git* repository, you can press `<ctrl-c>` to stop the container, then remove it. Your robot is now ready to be [deployed](../RunRobot.md).
