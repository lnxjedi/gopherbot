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
