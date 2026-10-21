# Daily Usage

Gopherbot listens the way a teammate does: it sees messages coming from users, channels, threads, and direct-message contexts, then routes them through command, ambient-message, and job-trigger logic.

In practice, most users mainly need to know four things:

- how to address the robot
- where a command is available
- how to discover commands with help
- how threads and direct messages affect replies

This section covers those user-facing behaviors rather than the internal implementation details.

## Stopping an AI conversation

If the optional AI plugin is enabled, address the robot with `stop-ai` in the
conversation thread, for example `;stop-ai`. It stops listening to that thread
and clears its saved conversation, even if an API error prevented the first
reply. A reply already in progress may still finish. In a direct message,
`stop-ai` clears the existing AI conversation memory.
