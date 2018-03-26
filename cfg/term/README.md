This configuration directory can be used for development and testing with a bot
named **Floyd** using the `terminal` connector, e.g.:

```
[./gopherbot]$ go build
[./gopherbot]$ ./gopherbot -l /tmp/bot.log -c cfg/term/
Terminal connector running; Use '|C<channel>' to change channel, or '|U<user>' to change user
c:general/u:alice -> floyd, info
c:general/u:alice -> 
general: Here's some information about my running environment:
The hostname for the server I'm running on is: bot.example.org
My install directory is: /home/username/go/src/github.com/lnxjedi/gopherbot
My local configuration directory is: cfg/term/
My software version is: Gopherbot v1.0.1-snapshot, commit: (manual build)
My alias is: ;
The administrators for this robot are: alice
c:general/u:alice -> ;quit
c:general/u:alice -> 
general: @alice Sayonara!
[./gopherbot]$
```