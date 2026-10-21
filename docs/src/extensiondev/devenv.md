# Container Dev Environment

The most straight-forward and widely available way to set up a development environment for your robot is to take advantage of the [gopherbot-dev](https://quay.io/repository/lnxjedi/gopherbot-dev?tab=info) container used in [Quick Setup](../botsetup/Plugin.md):

**1.** Create a new `<botname>-dev` directory and copy the robot's `.env` file to `<botname>-dev/environment`
Select `GOPHER_ENVIRONMENT=development` for local work, and put the intended
`PrimaryProtocol`, `DefaultProtocol`, and brain settings in development
configuration. Installed defaults no longer use `GOPHER_PROTOCOL` as a
launcher selector. The brain lock guards ownership of the selected brain;
development configuration should select the intended development data and
connector accounts explicitly.

**2.** From this directory, run the `gopherbot-dev` container, this time supplying the robot's environment. Using "clu" as an example:
```
$ docker run -p 127.0.0.1:3000:3000 --name clu-dev --rm --env-file environment quay.io/lnxjedi/gopherbot-dev:latest
```
> Note that you can also find a generic `run-robot.sh` script to use for this in [github.com/lnxjedi/gopherbot/resources/containers/dev](https://github.com/lnxjedi/gopherbot/blob/main/resources/containers/dev/run-robot.sh).

**3.** Now open your browser and connect to [http://127.0.0.1:3000](http://127.0.0.1:3000), where you'll be presented with the [Theia](https://github.com/eclipse-theia/theia) interface.

I find a three-pane layout most convenient; the top pane where code and configuration can be edited, and two terminal windows - one for running the robot in terminal mode, and one for running `gopherbot` CLI commands.

**4.** In a terminal window, run `gopherbot`. The bootstrap plugin will clone your robot's configuration repository, and start the robot in terminal mode:
```
$ gopherbot
2020/12/01 17:38:01 Initialized logging ...
...
general: Restore finished
OUT: unset SSH_AUTH_SOCK;
OUT: unset SSH_AGENT_PID;
OUT: echo Agent pid 625 killed;
c:general/u:alice ->
```

When you're finished with your robot, you can press `<ctrl-c>` to stop and remove the dev container, or from another window: `$ docker stop clu-dev`. Later sections will discuss how to push changes in this environment.
