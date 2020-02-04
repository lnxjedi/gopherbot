# Integrating with SSH

The **GopherCI** job(s) use *ssh* tasks for cloning repositories with public keys, and you can also use these tasks in your own pipelines for executing remote tasks.

## Configuring SSH

You start by choosing a passphrase for your robot's ssh keypair - make it something quite long; you shouldn't need to type it more than once. Use the `encrypt` command (normally with the terminal connector) to produce the encrypted value, and put it in a stanza like the following in your `robot.yaml`:

```yaml
ExternalTasks:
  "ssh-init":
    Parameters:
    - Name: BOT_SSH_PHRASE
      Value: {{ decrypt "xxxxx" }}
```

## Initializing

Once the robot knows it's passphrase, you can use the `generate keypair` administrator command to generate a new keypair, which will be stored in `$GOPHER_CONFIGDIR/ssh/`. The private key is encrypted with the robot's (also encrypted) passphrase, so this can be committed to the repository. The `pubkey` administrator command will display the robot's public key.

## Using in Pipelines

A pipeline using *ssh* might look something like this:
```bash
AddTask ssh-init

AddTask ssh-scan my.remote.host

AddTask exec ssh $SSH_OPTIONS user@my.remote.host "whoami"
```

* The `ssh-init` task will:
    * Start an `ssh-agent` and add the robot's key
    * Use `SetParameter` to store `SSH_AUTH_SOCK` and `SSH_AGENT_PID` in the pipeline
    * Set `SSH_OPTIONS` to e.g. `-F $GOPHER_CONFIGDIR/ssh/config` if the robot has a custom ssh config file
    * Add a **FinalTask** to kill the `ssh-agent` when the pipeline finishes
* `ssh-scan` insures a host is listed in `known_hosts`, if desired (this may be unneeded depending on the contents of `ssh/config`)
* The `exec` task calls ssh as normal, using `$SSH_OPTIONS` to pick up custom configuration if it exists

