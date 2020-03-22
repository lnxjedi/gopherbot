Dockerfiles for Gopherbot DevOps Chatbot running on Ubuntu, Amazon Linux, and CentOS

**NOTE: The `gopherbot-docker` repository is updated automatically by the build pipeline for `github.com/lnxjedi/gopherbot`, from the contents of `resources/docker`.

## Demo / Setup with the Default Robot

If you run a new **Gopherbot** container with no environment variables set, you'll get the *default robot*, **Floyd**, running the *terminal* connector. Floyd can tell knock-knock jokes, and can also kick-off setup with the **autosetup** plugin. To get started:
```shell
$ docker run -it --name ansfile lnxjedi/gopherbot:latest
Info: PID == 1, spawning child
Info: Starting pid 1 signal handler
Logging to robot.log; warnings and errors duplicated to stdout
Terminal connector running; Use '|c<channel|?>' to change channel, or '|u<user|?>' to change user
general: *******
general: Welcome to the *Gopherbot* terminal connector. Since no configuration was detected,
you're connected to 'floyd', the default robot.
general: If you've started the robot by mistake, just hit ctrl-D to exit and try
'gopherbot --help'; otherwise feel free to play around with the default robot - you can
start by typing 'help'. If you'd like to start configuring a new robot, type: ';setup
<protocol>'.
c:general/u:alice -> ;setup slack
general: Copy to answerfile.txt:
<-- snip answerfile.txt -->
...
<-- /snip -->

Edit your 'answerfile.txt' and run the container with '--env-file answerfile.txt'.
Exiting (press <enter> ...)
c:general/u:alice -> 
Terminal connector finished
# Remove the temporary container
$ docker container rm ansfile
```

After you've edited your `answerfile.txt`, run the container again and provide it to the container environment with `--env-file`:
```shell
$ docker run -it --env-file answerfile.txt lnxjedi/gopherbot:latest
Info: PID == 1, spawning child
Info: Starting pid 1 signal handler
Info: Logging to robot.log
null connector: Initializing encryption and restarting...
Info: Logging to robot.log
null connector: Continuing automatic setup...
...
```

... then follow the provided instructions to set yourself up as an administrator and save your robot's configuration to a *git* repository.

## Bootstrapping an existing robot

You can bootstrap an existing robot in to a docker container by simply running the container with a few environment variables. One possibility is to use a deploy key for the robot's custom repository; create an environment file named `environment` like so:
```shell
GOPHER_ENCRYPTION_KEY=ThisIsntReallyMyEncryptionKeySrsly
GOPHER_CUSTOM_REPOSITORY=git@github.com:parsley42/clu-gopherbot.git
GOPHER_PROTOCOL=slack
GOPHER_DEPLOY_KEY=-----BEGIN_OPENSSH_PRIVATE KEY-----:...:-----END_OPENSSH_PRIVATE_KEY-----:
```

Note that the deploy key is a _very_ long line, created with e.g.:
```shell
$ cat deploy_key | tr ' \n' '_:'
```

Put the `environment` file, along with the `Makefile` from this repository, in a directory with the same name as your robot; then you can start your robot using the `make` targets.
