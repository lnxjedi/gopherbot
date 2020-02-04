# Manual Setup
This section documents manual setup of a new robot custom configuration repository. Note that the documentation will often refer to a robot's *configuration repository*, even though using a *git* repository isn't strictly required.

> Note: This manual in general, and this section in particular, is not written as a complete step-by-step guide. Rather more of an outline, it skips a lot of e.g. `mkdir`, `sudo`, etc. If you're somewhat new to Linux systems administration, but have some experience with containers (e.g. [Docker](https://www.docker.com/)), you might get more from the chapter on [Setup with Docker](DockerSetup.md).

## 1. Create the `GOPHER_HOME` directory
Similar to an [Ansible](https://www.ansible.com/) playbook, a **Gopherbot** robot is heavily oriented around a standard directory structure for a given robot. To begin with, create an empty directory for your robot; `/var/lib/robots` or `/home/robots` are good places for this. I normally name the directory after the robot; in this example setup, we'll use `clu`:
```shell
$ 
```

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
2020/01/14 15:15:29 Warning: Starting unconfigured: stat custom/conf/robot.yaml: no such file or directory
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

### Editing Default Configuration
Now you need to edit the configuration files under `custom/`, replacing most of the `<replacevalue>` instances with values for your robot.

In `custom/conf/robot.yaml`:
* Replace `<defaultprotocol>` with "slack"
* Replace `<botname>` with the name of your robot, e.g. `clu`
* Replace `<botemail>` with an email address for your robot; will be used for "from:"
* Replace `<botfullname>` with a full name, informational only; e.g. "Clu Gopherbot"
* Replace `<botalias>` with a single-character alias for addressing your robot from the list `'&!;:-%#@~<>/*+^\$?\[]{}'` (";" is good for this)
* Replace `<sshencrypted>` encrypted ciphertext for a 16+ char ssh passphrase for your robot

In `custom/conf/slack.yaml`:
* Replace `<slackencrypted>` with the ciphertext generated above
* You'll replace `<adminusername>` and `<adminuserid>` later, after your robot has successfully connected

In `custom/git/config`, replace the values for `<botfullname>` and `<botemail>`; these will be used when your robot performs a `git commit`.

## SSH Keypair
Now generate a new ssh keypair for your robot, using the passphrase from above:
```shell
[clu]$ ssh-keygen -N "VeryLovelyNicePassword" -C "clu@linuxjedi.org" -f custom/ssh/robot_rsa
Generating public/private rsa key pair.
Your identification has been saved in custom/ssh/robot_rsa.
Your public key has been saved in custom/ssh/robot_rsa.pub.
The key fingerprint is:
...
```

## Connecting to Team Chat
Now your robot should be able to connect to team chat, and respond to your messages:
```shell
[clu]$ ./gopherbot 
2020/01/15 14:24:19 Initialized logging ...
2020/01/15 14:24:19 Loaded initial private environment from '.env'
2020/01/15 14:24:19 Starting up with config dir: custom, and install dir: /home/parse/git/gopher/gopherbot
...
2020/01/15 14:24:20 Info: Robot is initialized and running
```

Now you talk to your robot in the `#general` channel, and get the information you need from Slack to configure yourself as the robot's administrator:
```
#general me-> clu, whoami
#general clu-> You are 'Slack' user 'parsley/U0JLW8EMS', speaking in channel 'general/C0JLW8EP6', email address: parsley@linuxjedi.org
```

Given this reply, you would replace `<adminusername>` with `parsley` and `<adminuserid>` with `U0JLW8EMS` in `custom/conf/slack.yaml`. Now you can stop the robot with `ctrl-c` and restart with access to administrative commands.

## Saving Configuration and Updating your Robot

At this point the contents of `custom/` should be committed to a git repository, and your robot should be run with `GOPHER_CUSTOM_REPOSITORY=<clone-url>`. This allows for a common workflow where the robot is running in a container, VM, or server in your infrastructure, and updates to robot configuration are made by updating the robot's configuration repository, followed by an administrator `update` command, (e.g. `clu, update`). The `update` command will cause the robot to `git pull` it's custom configuration, then perform a reload and report any errors.
