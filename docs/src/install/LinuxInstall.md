# Build from Source

If you want the latest engine changes, build Gopherbot from a source checkout and keep that checkout as your install tree.

## Build steps

```bash
git clone https://github.com/lnxjedi/gopherbot.git
cd gopherbot
make gopherbot
```

That leaves you with a `gopherbot` binary in the repository root.

## Important detail

Do not copy just the binary somewhere else and throw the checkout away. The executable looks for shipped defaults and resources relative to its own location. A working source-based install tree includes at least:

- `gopherbot`
- `conf/`
- `lib/`
- `resources/`
- `plugins/`
- `jobs/`
- `tasks/`
- `robot.skel/`

## Typical source-based workflow

```bash
git clone https://github.com/lnxjedi/gopherbot.git ~/src/gopherbot
cd ~/src/gopherbot
make gopherbot

mkdir -p ~/robots/acme
cd ~/robots/acme
~/src/gopherbot/gopherbot
```

In that example:

- `~/src/gopherbot` is the engine install tree
- `~/robots/acme` is the robot home

That split is the normal v3 authoring workflow.
