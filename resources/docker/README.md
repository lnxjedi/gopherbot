Dockerfiles for Gopherbot Chatops robot for DevOps, running on Ubuntu, Amazon Linux, and CentOS

**NOTE: The `gopherbot-docker` repository is updated automatically by the build pipeline for `github.com/lnxjedi/gopherbot`, from the contents of `resources/docker`.

## Bootstrapping an existing robot

You can bootstrap an existing robot in to a docker container by simply running the container with a few environment variables. One possibility is to use a deploy key for the robot's custom repository; create an environment file like so:
```shell
GOPHER_ENCRYPTION_KEY=ThisIsntReallyMyEncryptionKeySrsly
GOPHER_CUSTOM_REPOSITORY=git@github.com:parsley42/clu-gopherbot.git
DEPLOY_KEY=-----BEGIN_OPENSSH_PRIVATE KEY-----:...:-----END_OPENSSH_PRIVATE_KEY-----:
```

Note that the deploy key is a _very_ long line, created with e.g.:
```shell
$ cat deploy_rsa | tr ' \n' '_:'
```

Put this environment file, along with the `Makefile` from this repository, in to a directory with the same name as your robot; then you can start your robot using the `make` targets.
