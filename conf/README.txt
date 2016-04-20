These are default/sample configuration files. Some of them require
tokens/credentials. To supply these, or override other configuration,
create your own configuration directory, copy files there and edit them,
and set the GOPHER_LOCALDIR environment variable to point to it.
If gobot-chatops finds a file there, it won't load the corresponding
file in this directory.

For example, virtually every connector protocol requires some kind
of credentials, so at the very least you will need to copy gopherbot.json
to $GOPHER_LOCALDIR and edit it.
