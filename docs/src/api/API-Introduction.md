# Robot API Overview

Every plugin, job, and task works through the Robot API.

## Main capability groups

- identity and attributes
- message sending and formatting
- prompting for replies
- memory and datum storage
- pipeline assembly
- admin and utility helpers

## API chapters

- [Language Choices](Languages.md)
- [Attribute Retrieval](Attribute-Retrieval-API.md)
- [Message Sending and Formatting](Message-Sending-API.md)
- [Prompting for Replies](Response-Request-API.md)
- [Memory and Brain Methods](Brain-API.md)
- [Pipeline Methods](Pipeline-API.md)
- [Utility Methods](Utility-API.md)
- [Miscellaneous Methods](Misc-Methods.md)

## Practical notes

- The current first-class extension languages in this manual are Go, Lua, JavaScript, Bash, Python, and Ruby.
- Not every language binding exposes every public Robot method. The detailed pages call out gaps where they matter.
- Connector-specific rendering details live in the connector appendices. For message formatting behavior that depends on Slack or another protocol, start with the API chapter, then check the relevant appendix.

## Important v3 behavior notes

- `AddCommand` composes work into the current pipeline; it does not fake a new user message.
- `AddJob` starts a child pipeline and does not automatically inherit every parent parameter.
- Prompt timeouts are longer on interactive local connectors such as SSH when using built-in interpreters or Go.
