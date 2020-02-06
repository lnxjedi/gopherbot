Dockerfiles for Gopherbot DevOps Chatbot running on Ubuntu, Amazon Linux, and CentOS

**NOTE: The `gopherbot-docker` repository is updated automatically by the build pipeline for `github.com/lnxjedi/gopherbot`, from the contents of `resources/docker`.

## Demo / Setup with the Default Robot

If you run a new **Gopherbot** container with no environment variables set, you'll get the *default robot*, **Floyd**, running the *terminal* connector. Floyd can tell knock-knock jokes, and can also run the **interactive setup plugin** which you can use to start configuring your own robot. To get started:
```shell
$  docker run -it lnxjedi/gopherbot
Info: PID == 1, spawning child
Debug: Successfully raised privilege permanently for 'exec child process' thread 1; new r/euid: 1/1
Info: Starting pid 1 signal handler
Debug: Checking os.Stat for dir 'custom' from wd '': stat custom: no such file or directory
Debug: Checking os.Stat for dir 'conf' from wd '': stat conf: no such file or directory
Warning: Starting unconfigured; no robot.yaml/gopherbot.yaml found
Terminal connector running; Use '|c<channel|?>' to change channel, or '|u<user|?>' to change user
Warning: GOPHER_CUSTOM_REPOSITORY not set, not bootstrapping
general: *******
general: Hi, I'm floyd, the default robot - I see you're running Gopherbot
unconfigured.
general: If you've started the robot by mistake, just hit ctrl-D to exit and
try 'gopherbot --help'; otherwise feel free to play around - you can start by
typing 'help'. If you'd like to start configuring a new robot, type:
';setup'.
```

## Bootstrapping an existing robot

You can bootstrap an existing robot in to a docker container by simply running the container with a few environment variables. One possibility is to use a deploy key for the robot's custom repository; create an environment file like so:
```shell
GOPHER_ENCRYPTION_KEY=ThisIsntReallyMyEncryptionKeySrsly
GOPHER_CUSTOM_REPOSITORY=git@github.com:parsley42/clu-gopherbot.git
GOPHER_PROTOCOL=slack
GOPHER_DEPLOY_KEY=-----BEGIN_OPENSSH_PRIVATE KEY-----:...:-----END_OPENSSH_PRIVATE_KEY-----:
```

Note that the deploy key is a _very_ long line, created with e.g.:
```shell
$ cat deploy_rsa | tr ' \n' '_:'
```

Put this environment file, along with the `Makefile` from this repository, in to a directory with the same name as your robot; then you can start your robot using the `make` targets.
