# Using the Setup Plugin

If you've installed the [Gopherbot software] on a Linux host or VM, you can just create an empty directory and run the `gopherbot` binary to start the default robot and use the **setup** plugin:

```
[~]$ mkdir clu
[~]$ cd clu/
[clu]$ ln -s /opt/gopherbot/gopherbot .
[clu]$ ./gopherbot 
2020/02/20 10:02:48 Debug: Checking os.Stat for dir 'custom' from wd '': stat custom: no such file or directory
2020/02/20 10:02:48 Debug: Checking os.Stat for dir 'conf' from wd '': stat conf: no such file or directory
2020/02/20 10:02:48 Warning: Starting unconfigured; no robot.yaml/gopherbot.yaml found
Terminal connector running; Use '|c<channel|?>' to change channel, or '|u<user|?>' to change user
c:general/u:alice -> 2020/02/20 10:02:49 Warning: GOPHER_CUSTOM_REPOSITORY not set, not bootstrapping
general: *******
general: Welcome to the *Gopherbot* terminal connector. Since no configuration was detected, you're
connected to 'floyd', the default robot.
general: If you've started the robot by mistake, just hit ctrl-D to exit and try 'gopherbot --help';
otherwise feel free to play around with the default robot - you can start by typing 'help'. If you'd
like to start configuring a new robot, type: ';setup'.
c:general/u:alice -> ;setup
...
```
