# Finding Commands with Help

The built-in help plugin is the fastest way to discover what a robot can do in your current context.

## Most useful commands

- `help`
- `help <keyword>`
- `commands`
- `help-all`
- `info`

## What makes v3 help better

Help in v3 is tied more closely to command metadata:

- `Usage`
- `Summary`
- `Examples`
- `Keywords`

That means users see help that is closer to the actual command surface rather than a detached block of prose.

## Context-aware behavior

Help is filtered by the current channel or DM context, and where possible by what the user should actually be allowed to see. If a command exists somewhere else, help can point you there instead of pretending the command does not exist.

## Advice for users

- Use `commands` when you want to browse.
- Use `help <keyword>` when you know roughly what you want.
- Use `info` when you need operational details about the robot itself.
