# Manual Setup
This section documents manual setup of a new robot custom configuration repository. Note that the documentation will often refer to a robot's *configuration repository*, even though using a *git* repository isn't strictly required.

## 1. Create the `GOPHER_HOME` directory
Similar to an [Ansible](https://www.ansible.com/) playbook, a **Gopherbot** robot is heavily oriented around a standard directory structure for a given robot. To begin with, create an empty directory for your robot. I normally name the directory after the robot; in this example setup, we'll use `clu`.

## 2. Create the `.env` file
Though it's possible to run **Gopherbot** with neither encryption nor a *git* repository, this example documents the more common scenario. To begin with, create a `.env` file in your new directory with contents similar to the following:
```shell
GOPHER_ENCRYPTION_KEY=SomeLongStringAtLeast32CharsOrItWillFail
```
Note that the robot will only use the first 32 chars of the encryption key, so no need to go crazy here. Keep your `GOPHER_ENCRYPTION_KEY` in a safe place, outside of any git repository. Your robot will need it, along with `$GOPHER_HOME/custom/binary-encrypted-key`, to decrypt it's secrets - such as a Slack token.

## 3. Initializing Encryption
Now that you've set the robot's encryption key, you can initialize encryption and create the binary key (`binary-encrypted-key`) encrypted with your `GOPHER_ENCRYPTION_KEY`. For convenience, go ahead and create a symlink for the **Gopherbot** binary in your `GOPHER_HOME`:
```shell
[clu]$ ln -s /opt/gopherbot/gopherbot .
```
Now start the default robot using the terminal connector, and the binary encryption key will be automatically created in `custom/binary-encrypted-key`; use `ctrl-d` to exit:

```
[clu]$ ./gopherbot 
2020/01/14 15:15:29 Warning: Starting unconfigured: stat custom/conf/gopherbot.yaml: no such file or directory
Terminal connector running; Use '|c<channel|?>' to change channel, or '|u<user|?>' to change user
2020/01/14 15:15:29 Warning: GOPHER_CUSTOM_REPOSITORY not set, not bootstrapping
general: Type ';setup' to continue setup...
c:general/u:alice -> <ctrl-d>
general: @alice Sayonara!
Terminal connector finished
[clu]$ ls custom/
binary-encrypted-key
```

## Initial Configuration
Now copy the contents of `robot.skel` from the distribution archive to the new `custom/` directory:
```shell
[clu]$ cp -a /opt/gopherbot/robot.skel/* custom/
```

### Encrypting your Slack Token (and other secrets)
With encryption initialized, you can use the `gopherbot encrypt` command to generate base64 encoded ciphertext. This ciphertext can then be placed in your configuration `yaml` files by using e.g. `{{ decrypt "jfhPe8akfivTRnfeCxyIetdUl+Jb7hIWnjeVFiwLJarFHuW4TuSD7GQ3F0s2puuZ3JUotw==" }}`. The default configuration assumes you'll only encrypt the portion of your slack token (from https://\<org\>.slack.com/services/new/bot) following the common `xoxb-` prefix. So, for a slack token of `xoxb-123-abc-XXXXX`, you would generate the ciphertext with:
```shell
[clu]$ ./gopherbot -l robot.log encrypt 123-abc-XXXXX
FverBqdWHzHfEPDy/cQ8U9AJ3z4v8KdGSubDMALPfHIupwDLctDWQ1c=
```
(Note that we use `-l robot.log` to send some superfluous error messages to a log file)

Now edit your `conf/slack.yaml` file and replace `<slackencrypted>` with the ciphertext, e.g.:
```yaml
  SlackToken: xoxb-{{ decrypt "FverBqdWHzHfEPDy/cQ8U9AJ3z4v8KdGSubDMALPfHIupwDLctDWQ1c=" }}
```

TODO: replace other `<foo>` strings, start robot and get admin username and ID.

## SSH Keypair

## Git Settings
