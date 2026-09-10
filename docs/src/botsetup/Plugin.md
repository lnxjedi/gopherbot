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

When you are ready to create your own robot, type `|c` to switch to a direct
conversation with the default robot. Then type `;new robot`. Onboarding accepts
the command only in a direct conversation with a validated administrator.

The working directory must not contain an existing Robot checkout. The
`custom/` path may be absent or completely empty; any file, hidden entry,
directory, or symlink there causes onboarding to stop without changing it.

## What happens next

Until a real robot repository is configured, you are talking to the default robot named `floyd`. Once `custom/` exists and your robot is configured, that local working directory becomes the robot.

Onboarding saves only restart and repository-boundary checkpoints in
`.setup-state`; it does not save individual questionnaire answers. If the
short questionnaire is interrupted, run `new robot` again or reconnect and
answer it from the beginning. After scaffold creation, repository handoff
resumes from its checkpoint. You do not need separate resume commands.
