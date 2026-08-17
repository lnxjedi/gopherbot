# Message Context

Gopherbot keeps track of more than just message text. Context can include:

- the user
- the protocol
- the channel
- the thread
- labeled values captured by command matchers

That context is what lets plugins do useful follow-up behavior without inventing their own ad-hoc state machine for every conversation.

## Examples

- prompting a user and waiting for the reply in the same channel or thread
- remembering the current list name or item in a short workflow
- continuing a threaded conversation without polluting the main channel

Context is intentionally scoped. In v3, thread-scoped memory includes protocol context so different connectors do not collide with each other.
