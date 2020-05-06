# The Built-in Help System

For a quick intro to the robot, the ambient[^1] command "help", by itself, will provide you with a quick overview of using the help system:
```
c:chat/u:alice -> help
chat: @alice I've sent you a private message introducing myself
(dm:alice): Hi, I'm Clu, a staff robot. I see you've asked for help.
...
```
Note that if you address the robot with it's name or alias, you instead get help for all the commands in the current channel:
```
c:chat/u:alice -> clu, help
chat: @alice (the help output was pretty long, so I sent you a private message)
(dm:alice): Command(s) available in channel: chat
Clu, help with robot - give general help on the help system and using the robot

Clu, help <keyword> - find help for commands matching <keyword>
...
```

**Gopherbot** ships with a simple keyword-based help system. Each plugin specifies a set of help texts linked to keywords, and these can easily be added to by providing custom `AppendHelp: ...` for a given plugin. Since **Gopherbot** commands are commonly linked to channels, keyword help will list the channels where a command is available if it's not in the current channel.

Additionally, some plugins may define custom "help with \<foo\>" commands that give extended help information about the plugin.

[^1]: "ambient" commands are matched against the entire message, every time. Once example of this is the "Chuck Norris" plugin; any time The Great One is mentioned, the robot will pipe up with an anecdote.