# Setup with Containers

If you have an environment where you can run and interact with containers in a terminal window, using either [Docker](https://www.docker.com/) or [Podman](https://podman.io/), you can supply an answerfile for using the **autosetup** plugin in a container.

## Getting `answerfile.txt`

For container-based installs, you'll have to download the appropriate answerfile for your protocol. These can be found in [resources/answerfiles](https://github.com/lnxjedi/gopherbot/tree/master/resources/answerfiles); save as `answerfile.txt`.

You can download directly with `wget`:
```shell
$ wget -O answerfile.txt https://raw.githubusercontent.com/lnxjedi/gopherbot/master/resources/answerfiles/slack.txt
```

Once you have your `answerfile.txt` you'll need to edit the file and set appropriate values for each `ANS_*` variable appropriate to your robot. Take care to use an unquoted value for `ANS_ROBOT_ALIAS`.

## Starting the *autosetup* plugin

With a complete answerfile, you can start a *Gopherbot* container, passing the answerfile in for the environment. After the *autosetup* plugin runs and generates configuration, it'll pause and provide instructions for completing setup.

**With Podman**:
```
$ podman run -it --env-file answerfile.txt lnxjedi/gopherbot:latest
Trying to pull docker.io/lnxjedi/gopherbot:latest...
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
$ docker run -it --env-file answerfile.txt lnxjedi/gopherbot:latest
Unable to find image 'lnxjedi/gopherbot:latest' locally
latest: Pulling from lnxjedi/gopherbot
...
Info: PID == 1, spawning child
Info: Starting pid 1 signal handler
Info: Logging to robot.log
null connector: Initializing encryption and restarting...
Info: Logging to robot.log
null connector: Continuing automatic setup...
...
```

## Finishing Up
After your robot has successfully saved it's configuration to it's *git* repository, you can press `<ctrl-c>` to stop the container, then remove it. Your robot is now ready to be [deployed](../RunRobot.md).