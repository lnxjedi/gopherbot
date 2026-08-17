# Connect Robot to Team
Now your robot should be able to connect to team chat, and respond to your messages:
```shell
davidparsley@penguin:~/git/clu$ GOPHER_PROTOCOL=slack ./gopherbot 
2020/02/19 15:50:00 Initialized logging ...
2020/02/19 15:50:00 Loaded initial private environment from '.env'
...
2020/02/19 15:50:20 Info: Robot is initialized and running
```

Now you can `/invite` your robot to the `#general` channel, and get the information you need from Slack to configure yourself as the robot's administrator:
```
#general me-> clu, whoami
#general clu-> You are 'Slack' user 'parsley/U0JLW8EMS', speaking in channel 'general/C0JLW8EP6', email address: parsley@linuxjedi.org
```

Given this reply, you would replace `<adminusername>` with `parsley` and `<adminuserid>` with `U0JLW8EMS` in `custom/conf/slack.yaml`. Now you can stop the robot with `ctrl-c` and restart with access to administrative commands.
