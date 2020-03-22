# Setting up a Robot with Gitpod

[Gitpod](https://gitpod.io) is an online development environment with a generous free offering for use with public free software projects. You can use **Gitpod** to configure a new robot, or to develop **Gopherbot** on your own fork. Once you've signed up for an account, you can use the **autosetup** plugin from the [Gitpod Online Demo](https://gitpod.io/#https://github.com/lnxjedi/gopherbot).

**1.** Use the `setup` command to create `answerfile.txt`:

```
############################################################################
Welcome to the Gopherbot Demo. This will run Gopherbot
in terminal connector mode, where you can use the 
autosetup plugin to configure a new robot and store it
in a git repository.
############################################################################

2020/03/22 18:25:29 Logging to robot.log; warnings and errors duplicated to stdout
Terminal connector running; Use '|c<channel|?>' to change channel, or '|u<user|?>' to change user
general: *******
general: Welcome to the *Gopherbot* terminal connector. Since no configuration
was detected, you're connected to 'floyd', the default robot.
general: If you've started the robot by mistake, just hit ctrl-D to exit and try
'gopherbot --help'; otherwise feel free to play around with the default robot -
you can start by typing 'help'. If you'd like to start configuring a new robot,
type: ';setup <protocol>'.
c:general/u:alice -> ;setup slack
general: Edit 'answerfile.txt' and re-run gopherbot with no arguments to
generate your robot.
Exiting (press <enter> ...)
c:general/u:alice -> 
Terminal connector finished
gitpod /workspace $
```

**2.** Open `answerfile.txt` in the left pane, and edit, taking note of the embedded documentation, and setting the "alias" per CLI-based setup.

**3.** Back in the **/workspace** tab, re-run `gopherbot` to continue setup:
```
gitpod /workspace $ ./gopherbot/gopherbot 
2020/03/22 18:38:46 Info: Logging to robot.log
null connector: Initializing encryption and restarting...
2020/03/22 18:38:48 Info: Logging to robot.log
null connector: Continuing automatic setup...
...
```

**4.** Follow the instructions given (you'll need to scroll back) to set yourself up as an administrator and save the robot's configuration to *git*. Note that the path to the **gopherbot** binary is `./gopherbot/gopherbot`.
