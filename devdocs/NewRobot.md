# New Robot Onboarding — Target Transcript

Design target for the redesigned `;new robot` flow. The user starts exactly one
setup command, `;new robot`, and the rest of the onboarding is resumed
automatically after reconnects. The target is two robot restarts total:

1. restart after writing `GOPHER_ENCRYPTION_KEY` to `.env`
2. final restart after local scaffold + git/bootstrap settings are complete

Bracketed `[notes]` annotate non-visible actions and implementation details so this transcript doubles as a
flow spec.

NOTES:
- All robot messages should be sent in BasicMarkdown format, preferably by converting the robot object to a formatted robot with `Format=BasicMarkdown`. The flow should take advantage of BasicMarkdown formatting to make it visually pleasant.
- The default robot's name and alias should be obtained by calls to `GetBotAttribute`.
- Reconnects should resume automatically through a join-triggered resume job; the user should not need to type `;new robot resume`, `;new robot repo`, or any other phase-transition command.
- The generated robot should keep a persistent encrypted SSH server host key in config, but the onboarding flow should not keep legacy `BOT_SSH_PHRASE` / self-managed outbound SSH key material from the old setup model.
- `go-new-robot` should not write `custom/binary-encrypted-key`; after the first restart establishes `GOPHER_ENCRYPTION_KEY`, the engine should create `custom/binary-encrypted-key` itself, and the onboarding flow should then use `EncryptSecret` for any needed encrypted config values.

---

## Sample identity assumptions

- The human in this transcript is **Samantha Jackson**.
- During the initial demo-mode setup, Samantha connects over SSH using the built-in demo user `alice` via `bot-ssh -d alice`.
- Samantha's local system account is `sjackson`, so auto-detected SSH keys come from paths under `/home/sjackson/`.
- Samantha's canonical robot username and team-chat identity is `samantha`.
- Sample git hosting values in this transcript use Samantha's account, for example `git@github.com:sjackson/clu-gopherbot.git`.

---

## Pre-conditions

- Gopherbot starts from a bare directory with no `custom/` and no `.env`.
- The SSH connector is active (default robot "floyd").
- Samantha connects via `bot-ssh -d alice` and lands in `#general`.

---

## Welcome (existing welcome-join job)

```
Floyd: @alice Welcome to the *Gopherbot* ssh connector! Since no configuration
       was detected, you're connected to 'floyd', the default robot.

Floyd: If you've started the robot by mistake, just hit ctrl-D to exit and try
       'gopherbot --help'; otherwise feel free to play around with the default
       robot - you can start by typing 'help'. If you'd like to start
       configuring a new robot, type: ';new robot'.
```

[Use one normal channel message per paragraph; greet the connecting user once in the first welcome paragraph]
[Use about a `0.5s` pause before the welcome and about a `2s` pause between successive descriptive paragraphs the user is expected to read]

---

## Stage 1 — Encryption bootstrap

User types `;new robot`.

```
Floyd: Let's build your robot together. First we'll create the one secret every
       robot needs: `GOPHER_ENCRYPTION_KEY`.

Floyd: This key protects secrets stored by the robot. It lives in `.env` or
       other environment variables supplied when the robot starts, stays
       outside git, and is one of the values you'll use again when deploying
       the robot somewhere else.

Floyd: Would you like me to generate a fresh key for you, or would you rather
       paste one you already have? (generate / supply)
```

### Path A — auto-generate (typical)

```
alice: generate

Floyd: All right. I'll generate one now and write it into `.env`.
```

[Robot generates a fresh `GOPHER_ENCRYPTION_KEY`]
[Robot writes `.env` containing only `GOPHER_ENCRYPTION_KEY=<key>`]

```
Floyd: Done. Your encryption key is now written to `.env`, which can be used
       for supplying environment variables to a robot when it starts.

Floyd: Keep `.env` safe and never commit it to git. After I restart, reconnect as
       @alice and we'll pick up automatically right where we left off.
```

[Robot writes `.setup-state` with `status=active`, stage=`awaiting-bot-name`]
[Robot removes any existing `custom/` scaffold and any existing `custom/binary-encrypted-key` from prior failed setup attempts before restarting]
[Robot adds `restart-robot` task to pipeline]

### Path B — user supplies key

```
alice: supply

Floyd: Please paste the encryption key you'd like to use.

alice: GretFilllLterbRounwn8-v]CarpeDie

Floyd: That looks valid, so I'll write it into `.env`.
```

[Robot validates the supplied key]
[Robot writes `.env` containing only `GOPHER_ENCRYPTION_KEY=<key>`]

```
Floyd: Done. Your encryption key is now written to `.env`, which can be used
       for supplying environment variables to a robot when it starts.

Floyd: Keep `.env` safe and never commit it to git. After I restart, we'll use
       this key for the rest of setup and I can stop doing the fiddly crypto
       work manually.

Floyd: I'm restarting now. Reconnect as @alice and I'll pick up automatically
       right where we left off.
```

[Robot writes `.setup-state` with `status=active`, stage=`awaiting-bot-name`]
[Robot removes any existing `custom/` scaffold and any existing `custom/binary-encrypted-key` from prior failed setup attempts before restarting]
[Robot adds `restart-robot` task to pipeline]

> Use the `GOPHER_USER` parameter (`GetParameter`) to determine the username.

---

## Engine restart (first)

[Gopherbot restarts and loads `GOPHER_ENCRYPTION_KEY` from `.env`]
[The engine sees `GOPHER_ENCRYPTION_KEY` and no `custom/binary-encrypted-key`, so it creates `custom/binary-encrypted-key` during normal encryption initialization]
[With matching `GOPHER_ENCRYPTION_KEY` and `custom/binary-encrypted-key` now established, the robot can use `EncryptSecret` for encrypted config values in later onboarding steps]
[SSH connector comes back up]

Samantha reconnects via `bot-ssh -d alice`.

At this point, the normal welcome job sees `.setup-state` and stays quiet.
The `resume-setup` join job detects the saved `.setup-state` and resumes automatically:

```
Floyd: Welcome back. I found onboarding progress in `.setup-state`, so I'll
       continue from where we left off.
```

---

## Stage 2 — Guided setup and repository handoff

### Robot name

```
Floyd: Your robot will need a 'given name' something like 'Floyd', but unique
       to *your* custom robot.

Floyd: For maximum compatibility and portability across chat platforms, the
       robot will recognize messages that start with its name as a command it
       should try to interpret, for example 'Floyd, ping'.

Floyd: What name would you like to give your robot?

alice: clu

Floyd: Perfect. Your robot will respond to messages that start with `clu`.
```

[Session updated: `botName=clu`, `stage=awaiting-bot-alias`]

### Robot alias

```
Floyd: Now let's pick a one-character alias for quick commands. If you choose
       ';', people can type ';ping' instead of 'Clu, ping'.

Floyd: I recommend ';', but you can choose any of these:
       ! ; - % ~ * + ^ $ ? [ ] { } \

alice: ;

Floyd: Sounds good. Your robot will also respond to `;`.
```

[Session updated: `botAlias=;`, `stage=awaiting-job-channel`]

### Job channel

```
Floyd: Your robot can run scheduled jobs, and it's helpful to give those job
       messages a default home.

Floyd: I suggest `clu-jobs`. If you'd like to use that, just reply `=`.
       Otherwise, send the channel name you'd prefer.

alice: =

Floyd: Got it. Scheduled job messages will go to `#clu-jobs`.
```

[Session updated: `jobChannel=clu-jobs`, `stage=awaiting-robot-email`]

### Robot email

```
Floyd: What email address should the robot use?

alice: samantha.jackson@example.com

Floyd: Thanks. The robot will use `samantha.jackson@example.com` for its own
       email identity.
```

[Session updated: `robotEmail=samantha.jackson@example.com`, `stage=awaiting-admin-email`]

### Administrator email

```
Floyd: What email address should the robot advertise for its administrator?

alice: samantha.jackson@example.com

Floyd: Understood. The robot will advertise
       `samantha.jackson@example.com` as its administrator contact.
```

[Session updated: `adminEmail=samantha.jackson@example.com`, `stage=awaiting-ssh-username`]

### Username and SSH key

```
Floyd: Right now you're connected as `alice`, which is one of the built-in demo
       users. Once your robot is configured, you'll connect as your actual robot
       username instead.

Floyd: If your robot will connect to team chat, you should supply the username you're
       known by there. What username would you like to use?

alice: samantha

Floyd: Good choice. You'll use `samantha` as your canonical username for this
       robot.
```

[Session updated: `canonicalUser=samantha`, `stage=awaiting-ssh-key`]

```
Floyd: I found this public key in your home directory:
       `/home/sjackson/.ssh/id_ed25519.pub`

Floyd: Would you like me to use that key for `samantha`? (y/n)

alice: y

Floyd: Excellent. I'll use the SSH public key from
       `/home/sjackson/.ssh/id_ed25519.pub` for local login.
```

[The home directory should come from the `$HOME` environment variable.]

[Session updated: `sshPublicKey=<key>`, `sshPublicKeySource=detected`]

### Scaffold creation

```
Floyd: That's everything I need for the local robot itself. I'll build the
       scaffold now, then we'll connect it to git before the final restart.
```

[Robot generates `custom/` from `robot.skel` with the collected values]
[Robot writes `custom/conf/robot.yaml` and supporting config]
[Robot writes SSH identity config for `samantha`]
[Robot saves the SSH server public key to `custom/robot-ssh.pub`]
[Robot writes an encrypted persistent SSH server host key into config for the SSH connector using `EncryptSecret`]
[Robot does not configure legacy `BOT_SSH_PHRASE` / self-managed outbound SSH key material for the new-robot path]
[Robot does not manually generate `custom/binary-encrypted-key`; that was already created by the engine on the first restart]
[Session updated: `stage=awaiting-repository-url`]

```
Floyd: I've created the local scaffold under `custom/` and configured a local
       SSH connector identity for `samantha`.

Floyd: The last setup step is configuring your robot's git repository for
       deployment bootstrapping.
```

### Repository handoff

```
Floyd: All of the custom configuration, scripts and secrets (encrypted by
       `GOPHER_ENCRYPTION_KEY`) that define your robot will be stored in a
       remote git repository.

Floyd: When `gopherbot` is started in an empty directory, or a directory
       containing only `.env`, it will look for the environment variables it
       needs there, clone your robot's repository, and then start your robot
       for simple deployment and updates.

Floyd: This should be a clone URL that uses SSH credentials. For this flow,
       the upstream repository should be created empty with no README, LICENSE,
       or other starter files.

Floyd: For example:
       `git@github.com:sjackson/clu-gopherbot.git`

alice: git@github.com:sjackson/clu-gopherbot.git

Floyd: Perfect. I'll configure bootstrap to use
       `git@github.com:sjackson/clu-gopherbot.git`.
```

[Robot generates an ed25519 deploy keypair]
[Robot writes deploy public key to `custom/ssh/deploy_key.pub`]
[Robot updates `.env` with `GOPHER_CUSTOM_REPOSITORY=<repo-url>`]
[Robot updates `.env` with `GOPHER_DEPLOY_KEY=<encoded-private-key>`]
[Robot updates `.env` with any other required bootstrap vars such as `GOPHER_ENVIRONMENT=development`]
[Session updated: repository handoff complete; setup can proceed without waiting for a typed confirmation]

```
Floyd: Repository handoff is ready. I've updated `.env` with the repository
       bootstrap settings and written the deploy public key to
       `custom/ssh/deploy_key.pub`.

Floyd: Add this read-only deploy key to your repository:

Floyd: `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExample... clu-deploy`

Floyd: When you're ready to publish the local scaffold, run these commands
       from the `custom/` directory:

       git init
       git add .
       git branch -m main
       git commit -m "Initial robot scaffold"
       git remote add origin git@github.com:sjackson/clu-gopherbot.git
       git push -u origin main

Floyd: After that first push succeeds, you should be able to start your robot
       by running `gopherbot` in a new empty directory with only `.env`.

Floyd: Meanwhile, I'll restart once more from the current directory, where
       your new robot resides in `custom/` - after the restart, you should be
       able to connect as yourself with `bot-ssh samantha`. Then, as
       administrator, you can start working on setting up a proper brain, a
       team chat connector, and other pieces needed for a fully functional
       robot.

Floyd: Have fun.
```

[Robot writes `.setup-state` with `status=completed`, stage=`repository-ready`]
[Robot adds `restart-robot` task to pipeline]

---

## Engine restart (final)

[Gopherbot restarts, loads `custom/` config, and uses the configured name/alias]
[Engine reuses the existing `custom/binary-encrypted-key` created on the first restart]
[SSH connector comes back up with the configured local user]

Samantha reconnects: `bot-ssh samantha`

```
Clu: I'm now running with your full robot configuration.

Clu: Your working directory is ready to keep using as the source repository
     checkout.

Clu: After you've created the empty upstream repository, added the deploy key,
     and pushed the local `custom/` checkout, test the bootstrap path from a
     brand-new empty directory somewhere else.

Clu: Copy only `.env` into that empty directory and start `gopherbot` there.

Clu: On first start, Gopherbot should read `.env`, clone
     `git@github.com:sjackson/clu-gopherbot.git`, build out `custom/`, and
     restart itself into the same fully configured robot.

Clu: If that works, you're done. You now have a robot that can be deployed by
     carrying only `.env` into an empty directory, or by starting the gopherbot
     engine in a new empty directory with the required environment variables
     already set.
```

---

## Summary of changes from current flow

| Aspect | Current | Target transcript |
|---|---|---|
| Entry flow | Setup splits across `;new robot`, reconnect, `;new robot repo`, and other resume commands | One setup command: `;new robot`; reconnects auto-resume without extra phase commands |
| `.env` bootstrap | Early flow writes multiple values and does extra setup-specific crypto work | First step writes only `GOPHER_ENCRYPTION_KEY`; later git/bootstrap vars are added when repo handoff is complete |
| Crypto handling | Onboarding code manually handles more key/encryption work, including legacy SSH key/passphrase setup | Keep the encrypted persistent SSH host key, remove legacy `BOT_SSH_PHRASE` baggage, let the engine generate `custom/binary-encrypted-key` on the first restart, and use `EncryptSecret` afterward |
| User prompts | Functional prompts, some terse and phase-oriented | Friendlier prompts that explain why each answer matters and offer defaults where helpful |
| Git handoff | Separate repository phase after scaffold restart | Repository setup is part of the same `;new robot` flow and explains the manual git steps without waiting for a typed `done` |
| Restarts | More than two reconnect moments in the draft transcript | Two restarts total: one after `.env` key creation, one final configured restart |
| Resume UX | User must type resume/repo commands after reconnect | Welcome flow resumes automatically for the active onboarding session |
| Bootstrap verification | Tells user to remove local files and restart in place | Tells user to copy `.env` into a new empty directory and test bootstrap there |
