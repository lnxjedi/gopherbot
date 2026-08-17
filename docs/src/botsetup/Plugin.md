# Run the Default Robot

When you start Gopherbot in an empty robot home, it does not fail. It starts the shipped default robot instead.

That default is meant to be useful, not just decorative:

- it starts the local SSH connector by default
- it includes built-in help and admin commands
- it can scaffold a new robot with `new robot`

## Start it

```bash
mkdir -p ~/robots/demo
cd ~/robots/demo
/opt/gopherbot/gopherbot
```

## Connect to it

```bash
chmod 600 /opt/gopherbot/resources/ssh-default/alice.key
ssh -i /opt/gopherbot/resources/ssh-default/alice.key -p 4221 alice@localhost
```

## First commands to try

- `help`
- `commands`
- `info`
- `whoami`
- `;new thread`
- `;new robot`

## What happens next

Until a real robot repository is configured, you are talking to the default robot named `floyd`. Once `custom/` exists and your robot is configured, that local working directory becomes the robot.

If you stop in the middle of onboarding, the engine keeps state in `.setup-state` and the `new robot resume` and `new robot repo` commands pick up where you left off.
