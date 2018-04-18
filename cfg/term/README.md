This configuration directory can be used for development and testing with a bot
named **Floyd** using the `terminal` connector, e.g.:

```
[gopherbot]$ go build -tags test
[gopherbot]$ ./gopherbot -c cfg/term/
2018/04/13 18:07:52 Initialized logging ...
2018/04/13 18:07:52 Starting up with config dir: cfg/term/, and install dir: /home/user/go/src/github.com/lnxjedi/gopherbot
2018/04/13 18:07:52 Debug: Loaded installed conf/gopherbot.yaml
2018/04/13 18:07:52 Debug: Loaded configured conf/gopherbot.yaml
Terminal connector running; Use '|C<channel>' to change channel, or '|U<user>' to change user
c:general/u:alice -> floyd, info
c:general/u:alice -> 
general: Here's some information about my running environment:
The hostname for the server I'm running on is: bot.example.org
My install directory is: /home/username/go/src/github.com/lnxjedi/gopherbot
My configuration directory is: cfg/term/
My software version is: Gopherbot v1.0.1-snapshot, commit: (manual build)
My alias is: ;
The administrators for this robot are: alice
c:general/u:alice -> ;quit
c:general/u:alice -> 
general: @alice Sayonara!
[gopherbot]$
```

**NOTE:** If you build gopherbot with `-tags test`, the terminal connector will
also show events emitted when you press `enter` on a blank line:

```
[gopherbot]$ go build -tags test
[gopherbot]$ ./gopherbot -l /tmp/bot.log -c cfg/term/
2018/04/13 18:07:52 Initialized logging ...
2018/04/13 18:07:52 Starting up with config dir: cfg/term/, and install dir: /home/user/go/src/github.com/lnxjedi/gopherbot
2018/04/13 18:07:52 Debug: Loaded installed conf/gopherbot.yaml
2018/04/13 18:07:52 Debug: Loaded configured conf/gopherbot.yaml
Terminal connector running; Use '|C<channel>' to change channel, or '|U<user>' to change user
c:general/u:alice -> ;ping
general: @alice PONG
c:general/u:alice ->
Events gathered: CommandPluginRan, GoPluginRan
c:general/u:alice -> ;quit
general: @alice Later gator!
```