## Quick Start with Docker
If you've already got a Docker host and Slack team, this is the absolute fastest way to get your very own robot up and running for playing around with, though not recommended for production installs.

1. Get a Slack token for your robot at https://\<your-team\>.slack.com/services/new/bot
1. Create the environment file `.env`, filling in your specific values (except for GOPHER_ADMIN, filled in later):
```env
GOPHER_ADMIN=
GOPHER_SLACK_TOKEN=xoxb-yourTokenHere
GOPHER_BOTNAME=nameOfYourBot
GOPHER_BOTFULLNAME="Optional Full Name"
# A one-character alias for addressing the robot, e.g. \ping
# Other good values: ; ! *
GOPHER_ALIAS=\
```
3. Start your robot with `docker run --env .env --name <nameOfYourBot> quay.io/lnxjedi/gopherbot`, for example:
```shell
$ docker run --env-file .env --name frodo quay.io/lnxjedi/gopherbot
```

If all goes well, your terminal window will start filling up with **Gopherbot** debug logs, and your robot will connect to your Slack team. Now you can invite your robot user to a channel and try out a few commands. Assuming you used `\` for the alias, and `frodo` for the robot's name, you can try:
* `\ping`
* `frodo, info`
* `help` - for general help
* `\help` - for help on available commands
* `\whoami`

### Adding an Administrator
These instructions assume you kept the default `\` for the robot's alias.

1. Use the `\whoami` command to determine your actual Slack username, which frequently differs from the display name you see in channels
1. Press \<ctrl-c\> to stop your container, then remove it:
```shell
$ docker container rm frodo
```
3. Edit the `.env` file, adding your Slack user name for `GOPHER_ADMIN`
1. Run the robot again as in step 3 above

Now you can try out some more commands:
* `\reload`
* `\info` (more verbose output for admins)

Additionally, there are a few commands you can try in a DM (private message) to the robot:
* `list plugins`
* `debug task memes`
* `quit` - the container will exit and need to be re-started
* `help` - the list of commands will include all the administrator commands

TODO: More documentation, including production installs.
