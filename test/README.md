This directory is used for **Gopherbot** integration tests.

These configuration sub-directories are used with the `test` connector for automated testing. See `*_test.go`. To run the robot with one of these configurations using the terminal connector, run `gopherbot` inside a configuration directory. Best to use the special build using `make testbot`. See `aidocs/TESTING_CURRENT.md` for the current integration harness overview.

# Developing Tests

* `make testbot` - creates a build of Gopherbot with special modifications for developing tests
* Run `gopherbot` in one of the sub-directories of `gopherbot/test`
* In another terminal tab, change to the same directory and `tail -f robot.log`; the `TEST/*` messages provide the same formatting
* At the terminal connector prompt, press `<enter>` once to drain collected events
* Send a message to the robot that should generate some kind of reply
* Press `<enter>` to collect events and indicate the end of an exchange
* Pipe the logs into `gopherbot/test/log_to_testcases.rb` to generate test cases

## Example Session
Here's a brief session; example robot interaction:
```
c:general/u:alice -> 
Terminal connector: Type '|c?' to list channels, '|u?' to list users, '|t?' for thread help
Events gathered: []Event{ScheduledTaskRan}
c:general/u:alice -> ;add bananas to the grocery list
LOG Debug: (lists) Retrieved lists config: &{global}
general: @alice I don't have a 'grocery' list, do you want to create it?
c:general/u:alice -> 
Terminal connector: Type '|c?' to list channels, '|u?' to list users, '|t?' for thread help
Events gathered: []Event{CommandTaskRan, GoPluginRan}
c:general/u:alice -> yes
general: Ok, I created a new grocery list and added bananas to it
c:general/u:alice -> 
Terminal connector: Type '|c?' to list channels, '|u?' to list users, '|t?' for thread help
Events gathered: []Event{}
```
> Note the first blank line clears the startup events, and the interaction starts with `;add bananas ...`

At this point I examine the log and see the last `TEST` items make up the interaction:
```
$ grep TEST/ robot.log | tail -6
2025/01/14 20:01:35 Info: TEST/INCOMING: aliceID, general, ";add bananas to the grocery list", false
2025/01/14 20:01:35 Info: TEST/OUTGOING: {alice, general, "I don't have a 'grocery' list, do you want to create it?", false}
2025/01/14 20:01:37 Info: TEST/EVENTS: []Event{CommandTaskRan, GoPluginRan}
2025/01/14 20:01:38 Info: TEST/INCOMING: aliceID, general, "yes", false
2025/01/14 20:01:38 Info: TEST/OUTGOING: {null, general, "Ok, I created a new grocery list and added bananas to it", false}
2025/01/14 20:01:39 Info: TEST/EVENTS: []Event{}
```

Then, I pipe that into `../log_to_testcases.rb` to generate the test cases:
```
$ grep TEST/ robot.log | tail -6 | ../log_to_testcases.rb 
        {aliceID, general, ";add bananas to the grocery list", false, []TestMessage{{alice, general, "I don't have a 'grocery' list, do you want to create it?", false}}, []Event{CommandTaskRan, GoPluginRan}, 0},
        {aliceID, general, "yes", false, []TestMessage{{null, general, "Ok, I created a new grocery list and added bananas to it", false}}, []Event{}, 0},
```

I added those to a new "TestDevel" in `test/bot_integration_test.go`, and then I run it:
```
$ TEST=Devel make test
go test -run Devel -v --tags 'test integration netgo osusergo static_build' -mod readonly -race ./test
=== RUN   TestDevel
    start_t.go:36: Initializing test bot with installpath: '/opt/gopherbot', configpath: '/opt/gopherbot/test/membrain' and logfile: /tmp/bottest.log
Removing temporary key: remove /opt/gopherbot/test/membrain/binary-encrypted-key: no such file or directory
    testMethods.go:41: Message sent to robot: &test.TestMessage{User:"u0001", Channel:"general", Message:";add bananas to the grocery list", Threaded:false, Hidden:false}
    testMethods.go:55: Reply received from robot: u:alice, c:general, m:I don't have a ' ..., t:false
    testMethods.go:41: Message sent to robot: &test.TestMessage{User:"u0001", Channel:"general", Message:"yes", Threaded:false, Hidden:false}
    testMethods.go:55: Reply received from robot: u:, c:general, m:Ok, I created a  ..., t:false
    testMethods.go:41: Message sent to robot: &test.TestMessage{User:"u0001", Channel:"", Message:"quit", Threaded:false, Hidden:false}
    connector.go:45: Received stop in connector
--- PASS: TestDevel (1.06s)
PASS
ok      github.com/lnxjedi/gopherbot/test       2.093s
```

*VOILA!*
