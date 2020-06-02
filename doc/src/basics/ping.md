# Addressing your Robot and using Ping

Most chat platforms provide some kind of capability for "mentioning" your robot using an `@...` syntax; normally, prefixing a message with your robot's mention name will cause it to process the message as a command. For maximum compatibility when switching chat platforms, **Gopherbot** robots all have a regular 'name' like 'Floyd' which they recognize, as well as a single-character 'alias'. To verify your robot "hears" messages, you would normally use the single "ping" command, which defaults to being available in all the channels where the robot is present. Here are some examples of using "ping" with **Floyd**, whose alias is `;`:
```
c:general/u:alice -> floyd, ping
general: @alice PONG
c:general/u:alice -> ;ping
general: @alice PONG
c:general/u:alice -> floyd ping
general: @alice PONG
c:general/u:alice -> floyd: ping
general: @alice PONG
c:general/u:alice -> ping, floyd
general: @alice PONG
c:general/u:alice -> ping
c:general/u:alice -> floyd
general: @alice PONG
c:general/u:alice -> ping
c:general/u:alice -> ;
general: @alice PONG
c:general/u:alice -> ping floyd
c:general/u:alice ->
```
The last three examples are instructive: 1) if you type a command for your robot but forget to address the robot, typing the robot's name or alias alone as your next message will cause it to process your previous message as a command if done within a short period of time. 2) While "\<command\>, \<botname\>" is considered a command, "\<command\> \<botname\>" (without a comma) is not; this allows the robot to be discussed (e.g. "try using floyd") without the robot parsing it as a command.