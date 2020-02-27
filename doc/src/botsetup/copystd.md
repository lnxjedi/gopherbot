# Copy and Modify the Standard Robot
Now copy the contents of `robot.skel` from the distribution archive to the new `custom/` directory:
```shell
davidparsley@penguin:/home/robots/clu$ cp -a /opt/gopherbot/robot.skel/* /opt/gopherbot/robot.skel/.??* custom/
```
> Note the `.??*` wildcard picks up any hidden files

### 4.1 Encrypting your Slack Token (and other secrets)
With encryption initialized, you can use the `gopherbot encrypt` command to generate base64 encoded ciphertext. This ciphertext can then be placed in your configuration `yaml` files by using e.g. `{{ decrypt "jfhPe8akfivTRnfeCxyIetdUl+Jb7hIWnjeVFiwLJarFHuW4TuSD7GQ3F0s2puuZ3JUotw==" }}`. The default configuration assumes you'll only encrypt the portion of your slack token (from https://\<org\>.slack.com/services/new/bot) following the common `xoxb-` prefix. So, for a slack token of `xoxb-123-abc-XXXXX`, you would generate the ciphertext with:
```shell
davidparsley@penguin:/home/robots/clu$ ./gopherbot -l robot.log encrypt 123-abc-XXXXX
FverBqdWHzHfEPDy/cQ8U9AJ3z4v8KdGSubDMALPfHIupwDLctDWQ1c=
```
(Note that we use `-l robot.log` to send some superfluous error messages to a log file)

Now edit your `conf/slack.yaml` file and replace `<slackencrypted>` with the ciphertext, e.g.:
```yaml
  SlackToken: xoxb-{{ decrypt "FverBqdWHzHfEPDy/cQ8U9AJ3z4v8KdGSubDMALPfHIupwDLctDWQ1c=" }}
```

### 4.2 Editing Standard Configuration
Now you need to edit the configuration files under `custom/`, replacing most of the `<replacevalue>` instances with values for your robot.

In `custom/conf/robot.yaml`:
* Replace `<botname>` with the name of your robot, e.g. `clu`
* Replace `<botemail>` with an email address for your robot; will be used for "from:"
* Replace `<botfullname>` with a full name, informational only; e.g. "Clu Gopherbot"
* Replace `<botalias>` with a single-character alias for addressing your robot from the list `'&!;:-%#@~<>/*+^\$?\[]{}'` (";" is good for this)
* Replace `<sshencrypted>` encrypted ciphertext for a 16+ char ssh passphrase for your robot, generated with `./gopherbot encrypt` as above

In `custom/conf/slack.yaml`:
* Leave alone for now; you'll replace `<adminusername>` and `<adminuserid>` later, after your robot has successfully connected

In `custom/conf/terminal.yaml`:
* Replace `<botalias>` with your robot's single-character alias

In `custom/git/config`, replace the values for `<botfullname>` and `<botemail>`; these will be used when your robot performs a `git commit`.
