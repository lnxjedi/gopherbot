# Standard Plugins and Commands

In addition to "ping", and "help", which we've already introduced, there are a few other standard commands normally available to all users:

* `info` - provides basic information about the robot's software version and where it's running:
```
c:chat/u:alice -> !info
chat: Here's some information about me and my running environment:
The hostname for the server I'm running on is: 
My name is 'Clu', alias '!', and my Terminal internal ID is '(unknown)'
This is channel 'chat', Terminal internal ID: #chat
The gopherbot install directory is: /home/davidparsley/git/gopherbot
My home directory ($GOPHER_HOME) is: /home/davidparsley/git/clu
My git repository is: git@github.com:parsley42/clu-gopherbot.git
My software version is: Gopherbot v2.0.0-beta3-snapshot, commit: dec5573
The administrators for this robot are: alice
The administrative contact for this robot is: David Parsley, <parsley@linuxjedi.org>
```
* `whoami` - gives information about how the robot "sees" you, with information useful for the `UserRoster`:
```
c:chat/u:alice -> !whoami
chat: You are 'Terminal' user 'alice/u0001', speaking in channel 'chat/#chat',
email address: alice@example.com
```

## The "links" Plugin
To simplify sharing common bookmarks with the rest of your team, you can use the "links" plugin:
```
c:general/u:alice -> !link employee manual to https://example.com/hr.html
general: Link added
c:general/u:alice -> !look up manual
general: Here's what I have for "manual":
https://example.com/hr.html: employee manual
c:general/u:alice -> !help with links
general: The links plugin stores URLs and associates them with a text key that can
be words or phrases. The 'link' command stores a link and key in one command, and the
'save' command will prompt the user to enter the key. The lookup command
will return all links whose key contains the provided word or phrase,
case insensitive. Links can be deleted with the 'remove' command.
```

## The "lists" Plugin
Slightly less useful, the "lists" plugin can be used for keeping simple lists of items:
```
c:general/u:alice -> !add beans to the grocery list
general: Ok, I added beans to the grocery list
c:general/u:alice -> !add milk to the list
general: Ok, I added milk to the grocery list
c:general/u:alice -> !show the list
general: Here's what I have on the grocery list:
bacon
tomatoes
bananas
wine
beer
mint cookies
salad
beans
milk
```

> Note that the last few commands simply referred to "the list", instead of fully specifying "the grocery list". Again, it's not AI; see the next section on "context".