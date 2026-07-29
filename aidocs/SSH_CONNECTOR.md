# SSH Connector Decisions

SSH is an authenticated collaborative chat transport, not merely a terminal
shim.

## Identity and startup

- Public keys map to canonical usernames through connector-local `UserKeys`.
  The list shape is deliberate: custom lists replace installed lists instead
  of recursively merging default identities.
- A successful configured key match sets `ValidatedUser=true`.
- The connector uses root `BotInfo` for robot identity; protocol-local bot-name
  duplication is intentionally absent.
- The listener probes the configured base port plus seven successors and writes
  the actual endpoint/public key to `.ssh-connect`.
- Reload atomically replaces key indexes and closes sessions whose key is no
  longer authorized. Listener address, port, and host key require restart.

## Privacy and routing

`/<botname> ...` is hidden and robot-addressed. A bare `/...` is hidden from
other SSH users but is not automatically addressed to the robot. This split
lets the engine enforce the same private-command policy used by app-command
transports.

User-to-user DMs are connector-local and never enter the engine. DMs with the
bot do. Hidden replies and DMs are viewer-scoped in replay/polling buffers; do
not weaken this when changing cursor or replay behavior.

Join announcements are bot-authored `SelfMessage` events. The engine may use
them for job triggers while excluding normal plugin matching.

## Terminal behavior

The replay buffer stores canonical plain text; optional live ANSI styling is a
terminal view, not API data. `BasicMarkdown` may retain source separately for
live rendering, but truncated buffers must not retain partial style state.
Direct/hidden message visibility applies to replay as well as live broadcast.

The local `golib/readline` fork adds bracketed-paste state and a pre-submit
display hook for timestamps. These hooks are intentional: multiline paste
should become one message, and display timestamps must not enter returned input
or history. Preserve fork tests when updating upstream readline behavior.
