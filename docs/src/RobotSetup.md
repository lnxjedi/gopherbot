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

On first run, no robot-specific config exists yet, so Gopherbot starts Floyd,
the fixed default Robot. By default it listens on the local SSH connector at
`localhost:4221`.

From another terminal:

```bash
ssh -i /opt/gopherbot/resources/ssh-default/alice.key -p 4221 alice@localhost
```

Once connected:

- run `help`
- run `info`
- type `|c` to switch to a direct conversation with the robot
- start the onboarding flow there with `;new robot`

The `new robot` flow requires a direct conversation with a validated administrator. It scaffolds a `custom/` tree, captures your initial SSH identity, and can later help with repository handoff so the robot is ready to bootstrap elsewhere.

Start onboarding only in a Robot home where `custom/` is absent or completely
empty. If that path contains anything—including hidden files, a `.git`
directory, or a symlink—the Robot refuses without changing it. Preserve or
move the existing data yourself before trying again.

The flow saves only the checkpoints needed around its two restarts and the
repository handoff. It does not persist questionnaire answers; if that short
part is interrupted, it starts again from the first question.

The generated scaffold keeps Robot identity and the default job channel in
`custom/conf/variables/common.yaml`. Its mostly static `robot.yaml` and SSH
connector configuration read those named values. Development- and
production-specific variables have documented files alongside `common.yaml`.
The persistent SSH server public key is written to
`custom/ssh-host-key.pub`; this is server identity, not an outbound Robot key.
New scaffolds do not include the deprecated terminal connector configuration.

The next few pages walk through that workflow in more detail.
