# New Robot Onboarding — Target Transcript

Design target for the redesigned `;new robot` flow. The flow is split into two
stages with an engine restart between them. Bracketed `[notes]` annotate
non-visible actions so this transcript doubles as a flow spec.

---

## Pre-conditions

- Gopherbot is running from a bare directory with no `custom/` and no `.env`.
- The SSH connector is active (default robot "floyd").
- Alice connects via `bot-ssh` and lands in `#general`.

---

## Welcome (existing welcome-join job)

```
Floyd: Welcome to the *Gopherbot* ssh connector, @alice! Since no configuration
       was detected, you're connected to 'floyd', the default robot.

Floyd: If you've started the robot by mistake, just hit ctrl-D to exit and try
       'gopherbot --help'; otherwise feel free to play around with the default
       robot - you can start by typing 'help'. If you'd like to start
       configuring a new robot, type: ';new robot'.
```

---

## Stage 1 — Encryption bootstrap

Alice types `;new robot`.

```
Floyd: Great, let's get a new robot set up! The first thing we need is an
       encryption key. Your robot uses a 32-character key to protect all its
       secrets — this key lives in a '.env' file outside of any git repository
       and should be kept somewhere safe.

Floyd: Would you like me to generate a random key, or do you have one you'd
       like to supply? (generate / supply)
```

### Path A — auto-generate (typical)

```
alice: generate

Floyd: Got it — generating a fresh encryption key now.
```

[Robot generates 32-char base64 key from /dev/random]
[Robot writes `.env` containing `GOPHER_ENCRYPTION_KEY=<key>`]

```
Floyd: I've written your encryption key to '.env'. *Keep this file safe and
       never commit it to git.* If you ever need to recover it, you'll find
       it printed below — copy it somewhere secure now:

Floyd: GOPHER_ENCRYPTION_KEY=GretFilllLterbRounwn8-v]CarpeDie

Floyd: Now I need to restart so the engine picks up the new key. When I come
       back, we'll continue setting up your robot — just reconnect and I'll
       pick up where we left off.
```

[Robot writes `.setup-state` with session status=active, stage=awaiting-bot-name]
[Robot adds restart-robot task to pipeline]

### Path B — user supplies key

```
alice: supply

Floyd: Paste your 32-character encryption key:

alice: GretFilllLterbRounwn8-v]CarpeDie

Floyd: Looks good.
```

[Robot validates key length/charset]
[Robot writes `.env` containing `GOPHER_ENCRYPTION_KEY=<key>`]

```
Floyd: I've written your key to '.env'. Now I need to restart so the engine
       picks up the new key. When I come back, we'll continue setup — just
       reconnect and I'll pick up where we left off.
```

[Robot writes `.setup-state` with session status=active, stage=awaiting-bot-name]
[Robot adds restart-robot task to pipeline]

---

## Engine restart

[Gopherbot restarts, loads GOPHER_ENCRYPTION_KEY from .env]
[SSH connector comes back up]

Alice reconnects via `bot-ssh`. The welcome-join job detects the saved
`.setup-state` and resumes:

```
Floyd: Welcome back, @alice! I found onboarding progress — let's keep going.
```

---

## Stage 2 — Guided robot setup

### Robot name

```
Floyd: First, what would you like to name your robot? This is the given name
       it'll recognize — people can get its attention with e.g. 'clu, ping'.

alice: clu

Floyd: Nice to meet Clu!
```

[Session updated: botName=clu, stage=awaiting-bot-alias]

### Robot alias

```
Floyd: Now pick a one-character alias for quick commands — for instance, if you
       choose ';' then ';ping' works as a shortcut. Pick one from:
       ! ; - % ~ * + ^ $ ? [ ] { } \

alice: ;

Floyd: Perfect, ';' it is.
```

[Session updated: botAlias=;, stage=awaiting-job-channel]

### Job channel

```
Floyd: Your robot may run scheduled jobs from time to time — backups, log
       rotation, that sort of thing. Output from those jobs goes to a default
       channel. A common convention is '<name>-jobs'. I'd suggest 'clu-jobs',
       or you can use 'general' if you don't expect many jobs. Reply '=' to
       use the suggestion.

alice: =

Floyd: Job output will go to #clu-jobs.
```

[Session updated: jobChannel=clu-jobs, stage=awaiting-email]

### Robot email

```
Floyd: What email address should your robot use? This is mostly informational
       to further identify the robot. If you don't have a dedicated address for the robot,
       your own email works fine.

alice: alice@example.com

Floyd: Got it — alice@example.com.
```

[Session updated: robotEmail=alice@example.com, stage=awaiting-ssh-key]

### Username and SSH key

```
Floyd: Right now you're connected as 'alice', one of the built-in demo users.
       Once your robot is configured, it won't have those built-in users
       anymore — you'll connect as yourself. So I need two things: the
       username you'd like to use, and the SSH public key you'll authenticate
       with.

Floyd: What username would you like? For team-chat robots, it's handy to use
       your team-chat username so everything lines up. Default 'alice'; reply
       '=' to use default.

alice: samantha

Floyd: Great, you'll connect as 'samantha'.
```

[Session updated: canonicalUser=samantha, stage=awaiting-ssh-key]

```
Floyd: Detected local SSH public key: /home/alice/.ssh/id_ed25519.pub — use
       that one? (y/n)

alice: y

Floyd: Got it — I'll configure that key for 'samantha' so you can connect with
       'bot-ssh -l samantha' after the restart.
```

[Session updated: sshPublicKey=<key>, sshPublicKeySource=detected]

### Scaffold generation

```
Floyd: All set! Let me build your robot's configuration scaffold now...
```

[Robot generates custom/ from robot.skel with all collected values]
[Robot generates binary-encrypted-key using GOPHER_ENCRYPTION_KEY]
[Robot writes custom/conf/robot.yaml with botName, alias, jobChannel, email]
[Robot writes custom/conf/slack.yaml (or terminal.yaml) connector stub]
[Robot writes SSH identity into custom/]
[Session updated: stage=scaffolded]

```
Floyd: Scaffold created under 'custom/' and local identity configured for
       'samantha'. I've also saved the SSH server public key to
       'custom/robot-ssh.pub'.

Floyd: Ok, I'll restart the robot so it picks up its new name and settings.
       Reconnect with 'bot-ssh -l samantha' and we'll finish up with the git
       repository.
```

[Robot adds restart-robot task to pipeline]

---

## Engine restart (second)

[Gopherbot restarts, loads custom/ config — now responds as "Clu" with alias ";"]

Alice reconnects: `bot-ssh -l samantha`

```
Clu: Welcome back, @samantha!

Clu: Type ';new robot repo' to continue where we left off.
```

---

## Stage 2b — Repository handoff

Alice types `;new robot repo`.

```
Clu: The last step is connecting your robot to a git repository. This is where
     all of Clu's configuration and scripts will live — you can bootstrap a
     fresh deployment just by pointing Gopherbot at this repo.

Clu: The repository should be empty (no README, LICENSE, or other files).
     What's the SSH clone URL? (e.g. 'git@github.com:alice/clu-gopherbot.git')

alice: git@github.com:alice/clu-gopherbot.git

Clu: Great — git@github.com:alice/clu-gopherbot.git.
```

[Robot generates ed25519 deploy keypair]
[Robot encrypts deploy private key, writes to .env as GOPHER_DEPLOY_KEY]
[Robot writes GOPHER_CUSTOM_REPOSITORY to .env]
[Robot writes deploy public key to custom/ssh/deploy_key.pub]
[Robot disables onboarding hooks in custom/conf/robot.yaml]
[Session updated: status=completed, stage=repository-ready]

```
Clu: Almost done! I've generated a deploy keypair and updated '.env'. Add this
     read-only deploy key to your repository:

Clu: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExample... clu-deploy

Clu: Now, from the 'custom/' directory, run these commands to push your robot's
     config:

     git init
     git add .
     git branch -m main
     git commit -m "New robot!"
     git remote add origin git@github.com:alice/clu-gopherbot.git
     git push -u origin main

Clu: Once that's done, let's test the bootstrap. From the robot's *parent*
     directory:

     1. Stop the robot (ctrl-C or 'kill <pid>')
     2. Remove everything except '.env':  rm -rf custom/ .setup-state
     3. Start gopherbot again

     Gopherbot will read '.env', see GOPHER_CUSTOM_REPOSITORY, clone your
     repo, and come up fully configured. If it works, you're all set!

Clu: Congratulations, @samantha — Clu is ready for duty. If you're planning to
     connect to Slack or another team chat, check the setup docs for connector
     configuration. And remember, that admin secret you saved earlier is how
     you'll register as admin on the new platform. Have fun!
```

---

## Summary of changes from current flow

| Aspect | Current | Redesigned |
|---|---|---|
| Encryption key | Handled externally via answerfile | Interactive Stage 1 prompt; robot writes .env itself |
| Engine restart | Single restart after scaffold | Two restarts: after .env creation, after scaffold |
| Robot email | Not collected | New prompt added |
| Admin secret | Not collected (external answerfile) | New prompt; robot generates/encrypts and stores |
| SSH key type | Collected via answerfile (ANS_KEY_TYPE) | Dropped — always ed25519 for deploy keys; user's own key type auto-detected |
| SSH passphrase | Collected via answerfile (ANS_SSH_PHRASE) | Dropped — deploy key is encrypted via GOPHER_ENCRYPTION_KEY, not a passphrase |
| Bootstrap test | Terse instructions | Friendly step-by-step with explanation of what happens |
| Tone | Functional/terse prompts | Conversational, explains *why* at each step |
| Resume | Session-aware resume | Same, plus post-restart auto-resume via welcome-join |
