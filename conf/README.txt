These are default/sample configuration files. Some of them require
tokens/credentials. To supply these, or override other configuration,
create your own configuration directory, copy files there and edit them,
and set the GOBOT_CONFIGDIR environment variable to point to it.
If gobot-chatops finds a file there, it won't load the corresponding
file in this directory.

For example, virtually every connector protocol requires some kind
of credentials, so at the very least you will need to copy gobot.json
to $GOBOT_CONFIGDIR and edit it.
