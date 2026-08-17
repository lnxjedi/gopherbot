# Administrative Commands

The built-in admin plugin provides the core operational command set.

## Most important commands

- `reload`
- `update`
- `switch-branch <branch>`
- `default-branch`
- `git-info`
- `protocol-list`
- `protocol-start <name>`
- `protocol-stop <name>`
- `protocol-restart <name>`
- `ps`
- `kill-pipeline <wid>`
- `pause-job <job>`
- `resume-job <job>`
- `paused-jobs`
- `quit`
- `restart`
- `abort`

## What these tell you

- branch commands answer "what config am I running?"
- protocol commands answer "which connectors are up right now?"
- process commands answer "what work is currently in flight?"

These commands are intentionally admin-only. They are part of the day-2 operational surface, not the normal user command set.
