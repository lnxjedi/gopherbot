# Command Matching and "command not found"

Like the UNIX command-line, your robot is sensitive to typos; more accurately, every robot command is checked against a set of regular expressions to see if a plugin is matched. It's not rocket science, and it's not AI - it's just good 'ol regexes. When you address your robot directly, but the message doesn't match a command regex, the robot's reply is a little more verbose than "command not found":
```
general: @alice Sorry, that didn't match any commands I know,
or may refer to a command that's not available in this channel;
try 'floyd, help <keyword>'
```
If you're sure you've typed the command correctly, your plugin may not be available in the current channel; the help system is useful for that:
```
c:chat/u:alice -> tell me a joke, clu
chat: @alice Sorry, that didn't match any commands I know,
or may refer to a command that's not available in this channel;
try 'Clu, help <keyword>'
c:chat/u:alice -> clu, help joke
chat: Command(s) matching keyword: joke
Clu, tell me a (knock-knock) joke (channels: general, random, botdev, clu-jobs)
c:chat/u:alice -> |cgeneral
Changed current channel to: general
c:general/u:alice -> tell me a joke, clu
general: Hang on while I Google that for you (just kidding ;-)
general: @alice Knock knock
c:general/u:alice -> Who's there?
general: @alice Weevil
c:general/u:alice -> Weevil who?
general: Weevil weevil rock you
```
