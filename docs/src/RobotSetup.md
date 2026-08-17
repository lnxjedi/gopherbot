# Create Your First Robot

The easiest way to understand Gopherbot is to treat the engine and the robot as two separate things:

- the engine is installed once
- a robot is a working directory plus git-managed custom config

In v3, you should usually build the engine once, create a fresh robot home, run the default robot locally, and let the onboarding flow scaffold `custom/` for you.

## Recommended first run

```bash
mkdir -p ~/robots/acme
cd ~/robots/acme
/opt/gopherbot/gopherbot
```

If you built from source instead of installing under `/opt/gopherbot`, use the path to that build tree's `gopherbot` binary instead.

On first run, no robot-specific config exists yet, so Gopherbot starts the default robot. By default it listens on the local SSH connector at `localhost:4221`.

From another terminal:

```bash
ssh -i /opt/gopherbot/resources/ssh-default/alice.key -p 4221 alice@localhost
```

Once connected:

- run `help`
- run `info`
- start the onboarding flow with `;new robot`

The `new robot` flow scaffolds a `custom/` tree, captures your initial SSH identity, and can later help with repository handoff so the robot is ready to bootstrap elsewhere.

The next few pages walk through that workflow in more detail.
