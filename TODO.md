# Upcoming tasks:

## Enabling MCP
Intermediate goal: updates to the engine and connectors for AI-based development.

### SSH Connector
* New default ssh connector default to replace terminal connector; this allows a local robot to run in the background and log to stdout, so a future gopherbot-mcp can start a local robot that the developer can connect to.
    * On connect, select initial filter:
        * A(ll) messages - you see every message from every channel and thread
        * C(hannel) messages - you see every message in the channel, including messages in every channel thread
        * T(hread) messages - if you've joined a thread, you see only messages in that thread, if you're in a channel, you only see messages sent to the channel without a thread

### Multi-protocol support
Backwards compat - old "protocol" is primary protocol, but the robot can connect to multiple transports. When a plugin runs, the protocol that triggered it is the protocol where all Say, Reply, etc. go; introduce new SendProtocolUserChannelThreadMessage for rare occassions. Job output goes to primary protocol. Admin command can change primary protocol in the event of e.g. Slack outage.

### Make built-in interpreters more powerful
Most functionality should be achievable with Lua, JavaScript or Go (yaegi) - certainly all *included* functionality, like protocol setup.

Implement DevOps helpers for JS/Lua (workspace-safe file ops + local exec wrappers, plus tests) — see checklists in `aidocs/JS_METHOD_CHECKLIST.md` and `aidocs/LUA_METHOD_CHECKLIST.md`. These are the basic methods needed by DevOps engineers to do most common automation tasks.