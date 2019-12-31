#  Developing Integration Tests

The current testing methodology for **Gopherbot** uses a special `test` protocol connector for sending test commands to the engine from various users in various channels, and then examining the responses and events generated. The tests themselves and test configurations are located in `/test`.

## Building the special "testbot"

The **Gopherbot** `Makefile` includes a special "testbot" target that builds the robot with a modified version of the terminal connector:

```shell
$ make testbot
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -tags 'netgo osusergo static_build test' -o gopherbot
```

## Developing Tests

See the contents of `test/*_test.go` for the format of the tests. After any given exchange with the robot, pressing `<enter>` by itself gives the events generated. Here's an example session for developing the tests in `test/bot_integration_test.go:TestPrompting`:

```shell
$ make testbot
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -tags 'netgo osusergo static_build test' -o gopherbot
[gopherbot]$ cd test/membrain
[membrain]$ ../../gopherbot 
Terminal connector running; Use '|c<channel|?>' to change channel, or '|u<user|?>' to change user
...
```
## Debugging Deadlocks

Occasionally locking changes in the core may result in deadlocks, which should hopefully be detected by the test suite. To assist in debugging these, the `SIGTERM` handler changes for the `test` build tag, causing the robot to perform a full stack dump and panic. Note that if an actual robot deadlocks, it should still accept an `abort` command from an administrator, which will trigger the same behavior.

If `make test` hangs, you can trigger a stack dump with `<ctrl-c>`; here's an example where updated locking of the `botContext` struct introduced a deadlock in the `reload` command:

```shell
$ make test
...
=== RUN   TestReload
# Test hung here, <ctrl-c>
goroutine 631 [running]:
github.com/lnxjedi/gopherbot/bot.sigHandle(0xc0000a1f20)
        /home/davidparsley/git/gopherbot/bot/signal_testing.go:24 +0x243
created by github.com/lnxjedi/gopherbot/bot.run
        /home/davidparsley/git/gopherbot/bot/bot_process.go:310 +0x3e4
...
goroutine 590 [semacquire]:
sync.runtime_SemacquireMutex(0xc000358504, 0x900000000, 0x1)
        /usr/local/go/src/runtime/sema.go:71 +0x47
sync.(*Mutex).lockSlow(0xc000358500)
        /usr/local/go/src/sync/mutex.go:138 +0x1c1
sync.(*Mutex).Lock(0xc000358500)
        /usr/local/go/src/sync/mutex.go:81 +0x7d
github.com/lnxjedi/gopherbot/bot.Robot.getLockedContext(0xc00016e540, 0x81, 0x0)
        /home/davidparsley/git/gopherbot/bot/robot.go:25 +0x4e
github.com/lnxjedi/gopherbot/bot.Robot.Reply(0xc00016e540, 0x81, 0xbf0ee6, 0x23, 0x0, 0x0, 0x0, 0x0)
        /home/davidparsley/git/gopherbot/bot/robot_connector_methods.go:212 +0x121
github.com/lnxjedi/gopherbot/bot.admin(0xce2020, 0xc0001ff930, 0xc0000b6320, 0x6, 0xc0001fedb0, 0x0, 0x0, 0x0)
        /home/davidparsley/git/gopherbot/bot/builtins.go:345 +0x16b2
github.com/lnxjedi/gopherbot/bot.(*botContext).callTaskThread(0xc000358500, 0xc0003df6e0, 0xb0ef20, 0xc00012a180, 0xc0000b6320, 0x6, 0xc0001fedb0, 0x0, 0x0)
        /home/davidparsley/git/gopherbot/bot/calltask.go:138 +0xb64
created by github.com/lnxjedi/gopherbot/bot.(*botContext).callTask
        /home/davidparsley/git/gopherbot/bot/calltask.go:75 +0xe8
...
panic: Tests terminated by signal terminated

goroutine 631 [running]:
github.com/lnxjedi/gopherbot/bot.sigHandle(0xc0000a1f20)
        /home/davidparsley/git/gopherbot/bot/signal_testing.go:27 +0x37a
created by github.com/lnxjedi/gopherbot/bot.run
        /home/davidparsley/git/gopherbot/bot/bot_process.go:310 +0x3e4
FAIL    github.com/lnxjedi/gopherbot/test       58.197s
FAIL
Makefile:52: recipe for target 'test' failed
make: *** [test] Error 1
```

In this case the lock on the `botContext` was first aquired at the top of the admin `reload` built-in command, and then hung later when trying to acquire the lock in the robot `Reply` method.