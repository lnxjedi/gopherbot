# Install the Engine

If you prefer a cleaner separation between the engine and your robot homes, unpack a release archive into a stable location such as `/opt/gopherbot`.

## Suggested layout

```text
/opt/gopherbot/
  gopherbot
  conf/
  lib/
  resources/
  plugins/
  jobs/
  tasks/
  robot.skel/
```

## Running a robot from that install

Create a robot home somewhere writable:

```bash
mkdir -p ~/robots/acme
cd ~/robots/acme
/opt/gopherbot/gopherbot
```

That command uses:

- the install tree at `/opt/gopherbot` for built-in assets
- the current directory for writable robot state and `custom/`

## Sample SSH key for the default robot

The shipped default robot includes sample SSH users and keys so you can connect locally without doing any setup first. For the `alice` demo user:

```bash
chmod 600 /opt/gopherbot/resources/ssh-default/alice.key
ssh -i /opt/gopherbot/resources/ssh-default/alice.key -p 4221 alice@localhost
```

That is the fastest way to kick the tires on a fresh install.
